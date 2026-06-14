// Package psiphon_v2ray là entry point cho gomobile bind.
// Build thành AAR dùng:
//   gomobile bind -target=android -o psiphon_v2ray.aar github.com/yourname/psiphon_v2ray_aar/psiphon_v2ray
//
// AAR này export Java package "psiphon_v2ray" với:
//   - V2Ray API: Libpsv2ray (init, newCoreController)
//   - Psiphon API: startPsiphon / stopPsiphon
package psiphon_v2ray
