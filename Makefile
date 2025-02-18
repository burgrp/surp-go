wireshark-install:
	mkdir -p ${HOME}/.local/lib/wireshark/plugins
	ln -s ${PWD}/wireshark/surp.lua ${HOME}/.local/lib/wireshark/plugins/surp.lua

wireshark-uninstall:
	rm -f ${HOME}/.local/lib/wireshark/plugins/surp.lua