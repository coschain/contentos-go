rm -f *.wasm
rm -f *.wast
rm -f *.abi

cosiocc -o has_floats.wasm has_floats.cpp
cosiocc -g has_floats.abi has_floats.cpp

cosiocc -o hello.wasm hello.cpp
cosiocc -g hello.abi hello.cpp

cosiocc -o hello2.wasm hello2.cpp
cosiocc -g hello2.abi hello2.cpp

cosiocc -o limits.wasm limits.cpp
cosiocc -g limits.abi limits.cpp

cosiocc -o native_tester.wasm native_tester.cpp
cosiocc -g native_tester.abi native_tester.cpp

cosiocc -o unknown_imports.wasm unknown_imports.cpp
cosiocc -g unknown_imports.abi unknown_imports.cpp

cosiocc -o gas_burner.wasm gas_burner.cpp burners.cpp
cosiocc -g gas_burner.abi gas_burner.cpp
