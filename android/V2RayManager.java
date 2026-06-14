package com.unlimited.vpn.manager;

import android.content.Context;
import android.net.VpnService;
import android.util.Log;

// Import từ psiphon_v2ray.aar (AAR mới build bằng gomobile)
import psiphon_v2ray.CoreCallbackHandler;
import psiphon_v2ray.CoreController;
import psiphon_v2ray.PsiphonController;
import psiphon_v2ray.PsiphonNoticeHandler;
import psiphon_v2ray.Psiphon_v2ray;  // top-level functions
import psiphon_v2ray.SocketProtector;

/**
 * V2RayManager - sử dụng API mới của psiphon_v2ray.aar.
 *
 * API mới (mirror libv2ray1):
 *   - Psiphon_v2ray.initCoreEnv(assetPath, userPath)
 *   - Psiphon_v2ray.newCoreController(handler) → CoreController
 *   - controller.startLoop(configJSON, tunFd)
 *   - controller.stopLoop()
 *   - controller.queryStats(tag, direction)
 *
 * Psiphon API:
 *   - Psiphon_v2ray.newPsiphonController(handler) → PsiphonController
 *   - psiphonCtrl.startTunnel(configJSON)
 *   - psiphonCtrl.stopTunnel()
 *   - psiphonCtrl.getSocksPort()
 */
public class V2RayManager {

    private static final String TAG = "V2RayManager";
    private static volatile V2RayManager instance;

    private Context context;
    private VpnService vpnService;
    private CoreController coreController;
    private PsiphonController psiphonController;
    private boolean envInitialized = false;

    private V2RayManager() {}

    public static V2RayManager getInstance() {
        if (instance == null) {
            synchronized (V2RayManager.class) {
                if (instance == null) instance = new V2RayManager();
            }
        }
        return instance;
    }

    // =========================================================
    // Init
    // =========================================================

    public void init(Context context, VpnService vpnService) {
        this.context = context.getApplicationContext();
        this.vpnService = vpnService;

        if (!envInitialized) {
            // Set socket protector để V2Ray protect socket khỏi VPN loop
            Psiphon_v2ray.setSocketProtector(new SocketProtector() {
                @Override
                public boolean protectFd(int fd) {
                    return V2RayManager.this.vpnService != null
                        && V2RayManager.this.vpnService.protect(fd);
                }
            });

            // Init V2Ray environment
            String assetPath = context.getFilesDir().getAbsolutePath();
            String userPath  = context.getFilesDir().getAbsolutePath();
            Psiphon_v2ray.initCoreEnv(assetPath, userPath);

            envInitialized = true;
            Log.d(TAG, "V2Ray env initialized");
        }
    }

    public void updateVpnService(VpnService vpnService) {
        this.vpnService = vpnService;
    }

    // =========================================================
    // V2Ray control
    // =========================================================

    /**
     * Khởi động V2Ray với config JSON và TUN fd.
     *
     * @param configJSON  Xray config JSON string
     * @param tunFd       TUN fd từ VpnService.Builder.establish(), hoặc -1
     * @param callback    Callback nhận status từ V2Ray
     */
    public void startV2Ray(String configJSON, int tunFd, V2RayCallback callback) {
        stopV2Ray(); // đảm bảo không có instance cũ

        coreController = Psiphon_v2ray.newCoreController(new CoreCallbackHandler() {
            @Override
            public long onEmitStatus(long code, String status) {
                Log.d(TAG, "V2Ray status: " + status);
                if (callback != null) callback.onStatus(status);
                return 0;
            }

            @Override
            public long shutdown() {
                Log.i(TAG, "V2Ray shutdown");
                if (callback != null) callback.onDisconnected();
                return 0;
            }

            @Override
            public long startup() {
                Log.i(TAG, "V2Ray startup");
                if (callback != null) callback.onConnected();
                return 0;
            }
        });

        try {
            coreController.startLoop(configJSON, tunFd);
            Log.i(TAG, "V2Ray startLoop OK, tunFd=" + tunFd);
        } catch (Exception e) {
            Log.e(TAG, "startLoop failed", e);
            if (callback != null) callback.onError(e.getMessage());
            coreController = null;
        }
    }

    public void stopV2Ray() {
        if (coreController != null) {
            try {
                coreController.stopLoop();
            } catch (Exception e) {
                Log.e(TAG, "stopLoop error", e);
            }
            coreController = null;
        }
    }

    public boolean isV2RayRunning() {
        return coreController != null && coreController.getIsRunning();
    }

    public long queryV2RayStats(String tag, String direction) {
        if (coreController == null) return 0L;
        return coreController.queryStats(tag, direction);
    }

    // =========================================================
    // Psiphon control
    // =========================================================

    /**
     * Khởi động Psiphon tunnel.
     *
     * @param psiphonConfigJSON  Psiphon config JSON
     * @param callback           Callback nhận status từ Psiphon
     */
    public void startPsiphon(String psiphonConfigJSON, PsiphonCallback callback) {
        stopPsiphon();

        psiphonController = Psiphon_v2ray.newPsiphonController(new PsiphonNoticeHandler() {
            @Override
            public void onNotice(String noticeJSON, long timestamp, boolean isDiagnostic) {
                Log.v(TAG, "Psiphon notice: " + noticeJSON);
                if (callback != null) callback.onNotice(noticeJSON);

                // Parse SOCKS port
                try {
                    org.json.JSONObject obj = new org.json.JSONObject(noticeJSON);
                    String type = obj.optString("noticeType", "");
                    if ("ListeningSocksProxyPort".equals(type)) {
                        int port = obj.optJSONObject("data").optInt("port", 0);
                        if (port > 0 && callback != null) {
                            callback.onSocksProxyReady(port);
                        }
                    } else if ("Tunnels".equals(type)) {
                        int count = obj.optJSONObject("data").optInt("count", 0);
                        if (count > 0 && callback != null) {
                            callback.onConnected();
                        }
                    } else if ("Exiting".equals(type)) {
                        if (callback != null) callback.onDisconnected();
                    }
                } catch (Exception ignored) {}
            }
        });

        try {
            psiphonController.startTunnel(psiphonConfigJSON);
            Log.i(TAG, "Psiphon startTunnel OK");
        } catch (Exception e) {
            Log.e(TAG, "startTunnel failed", e);
            if (callback != null) callback.onError(e.getMessage());
            psiphonController = null;
        }
    }

    public void stopPsiphon() {
        if (psiphonController != null) {
            try {
                psiphonController.stopTunnel();
            } catch (Exception e) {
                Log.e(TAG, "stopTunnel error", e);
            }
            psiphonController = null;
        }
    }

    public boolean isPsiphonRunning() {
        return psiphonController != null && psiphonController.getIsRunning();
    }

    public int getPsiphonSocksPort() {
        if (psiphonController == null) return 0;
        return psiphonController.getSocksPort();
    }

    // =========================================================
    // Cleanup
    // =========================================================

    public void destroy() {
        stopV2Ray();
        stopPsiphon();
        context = null;
        vpnService = null;
    }

    // =========================================================
    // Callback interfaces
    // =========================================================

    public interface V2RayCallback {
        void onConnected();
        void onDisconnected();
        void onStatus(String status);
        void onError(String error);
    }

    public interface PsiphonCallback {
        void onConnected();
        void onDisconnected();
        void onSocksProxyReady(int socksPort);
        void onNotice(String noticeJSON);
        void onError(String error);
    }
}
