rd /s /q "gen-go"
thrift-0.11.0.exe --gen go casperproto.thrift
xcopy "gen-go" "." /e/Y
rd /s /q "gen-go"