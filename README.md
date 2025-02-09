# surp-go
SURP (Simple UDP Register Protocol) implementation in Go


socat -v UDP6-RECV:5070,ipv6-join-group=[ff02::1fc6:1:1]:wlp3s0,reuseaddr -
socat - UDP6-SENDTO:[ff02::1fc6:1:1%wlp3s0]:5070

- implement Filtered socket
- unit tests


## Wireshark
WIRESHARK_PLUGIN_DIR=$PWD/wireshark wireshark
