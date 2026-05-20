# Source Anchors v0.2

These anchors are not full documentation mirrors. They are provenance anchors for Codex and future human review.

- `apple_networksetup`: Apple Remote Desktop guide states `networksetup` is used to configure client network settings and exposes service/proxy/network commands. https://support.apple.com/guide/remote-desktop/about-networksetup-apdd0c5a2d5/mac
- `apple_proxy_settings`: Apple Mac User Guide describes system proxy settings including auto proxy discovery, PAC, HTTP, HTTPS, SOCKS, simple-hostname exclusion, and bypass domains. https://support.apple.com/guide/mac-help/change-proxy-settings-on-mac-mchlp2591/mac
- `curl_proxy_env`: everything curl documents proxy environment variables and NO_PROXY behavior. https://everything.curl.dev/usingcurl/proxies/env.html
- `git_proxy_config`: Git config documentation includes HTTP proxy configuration behavior. https://git-scm.com/docs/git-config
- `npm_proxy_config`: npm config documentation includes proxy/https-proxy and environment proxy behavior. https://docs.npmjs.com/cli/v9/using-npm/config/
- `homebrew_env_docs`: Homebrew manpage documents Homebrew environment files and environment-variable behavior. https://docs.brew.sh/Manpage
- `docker_proxy_docs`: Docker documents Docker CLI/container/build proxy configuration. https://docs.docker.com/engine/cli/proxy/
- `docker_desktop_proxy_docs`: Docker Desktop settings documentation covers proxy settings and managed settings behavior. https://docs.docker.com/desktop/settings-and-maintenance/settings/
- `openai_codex_config`: OpenAI Codex configuration docs cover config files and shell environment policy. https://developers.openai.com/codex/config-reference and https://developers.openai.com/codex/config-advanced
- `clash_verge_macos_tun`: Clash Verge macOS FAQ notes that TUN changes system DNS and restores it when TUN is closed. https://www.clashverge.dev/faq/macos.html
- `audinate_dante_clock`: Audinate Dante Controller docs describe Dante clock synchronization via IEEE 1588 PTP and leader clock behavior. https://dev.audinate.com/GA/dante-controller/userguide/webhelp/content/clock_synchronization.htm
- `audinate_aes67_ptpv2`: Audinate Clock Config docs state AES67/ST 2110-30 require PTPv2 for accurate clock synchronization. https://dev.audinate.com/GA/dante-controller/userguide/webhelp/content/clock_config_tab.htm
- `ravenna_aes67_igmp`: RAVENNA AES67 practical guide discusses multicast, managed switches, IGMP snooping, and IGMPv2 requirements. https://www.ravenna-network.com/category/aes67-practical-guide/
- `macos_firewall_pf`: macOS local command/manpage knowledge; verify against `man pfctl`, Application Firewall UI, and third-party firewall documentation in implementation.
- `tls_runtime_docs`: Runtime-specific trust behavior must be verified per curl/Python/Node/Java/OpenSSL/certifi docs during implementation.
- `developer_tool_docs`: Tool-specific docs should be checked by Codex at implementation time for the exact installed version.
- `external_baseline`: External status/provider checks are evidence sources, not local repair sources.
