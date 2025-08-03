wireshark-install:
	mkdir -p ${HOME}/.local/lib/wireshark/plugins
	ln -s ${PWD}/wireshark/surp.lua ${HOME}/.local/lib/wireshark/plugins/surp.lua

wireshark-uninstall:
	rm -f ${HOME}/.local/lib/wireshark/plugins/surp.lua

proto:
	mkdir -p pkg/pb
	protoc --go_out=. --go_opt=paths=source_relative proto/surp.proto
	mv proto/surp.pb.go pkg/pb/surp.pb.go
