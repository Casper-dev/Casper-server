# Приложение для поставщика CasperAPI.

## Инструкции по сборке и установке

### Windows-платформы

Далее подразумевается, что заданы переменные среды GOROOT и GOPATH, а в переменную PATH добавлено %GOPATH%/bin

Скачать зависимости
```
go get -u -d -v gitlab.com/casperDev/contractsCasperApi
go get -u -d -v gitlab.com/casperDev/Casper-thrift
go get -u -d -v gitlab.com/casperDev/Casper-SC
go get -u -d -v github.com/whyrusleeping/gx
go get -u -d -v github.com/whyrusleeping/gx-go
go get -u -d -v github.com/ethereum/go-ethereum
go get -v git.apache.org/thrift.git/lib/go/thrift/...
```

И скачать бинарники solc (не забыв дополнить ими PATH) из https://github.com/ethereum/solidity/releases


После этого ставим все необходимые ipfs зависимости через gx
```
cd %GOPATH%/src/github.com/ipfs/go-ipfs
gx --verbose install --global
```

Если gx или gx-go не установились автоматически нужно выполнить
```
cd %GOPATH%/src/github.com/whyrusleeping/gx
go install
cd %GOPATH%/src/github.com/whyrusleeping/gx-go
go install
```
Собираем последнюю версию thrift-протокола
```
cd %GOPATH%/src/gitlab.com/casperDev/Casper-thrift
gen.bat
```
Генерируем последнюю версию контракта для вызовов
```
cd %GOPATH%/src/gitlab.com/casperDev/contractsCasperApi
abigen -sol contracts/Casper.sol -pkg casper > ../Casper-SC/casper_sc/casper_sc.go
```

Если abigen не устанился при загрузке, выполнить
```
cd %GOPATH%/src/github.com/ethereum/go-ethereum/cmd/abigen
go install
```

Собрать ipfs используя go install
```
cd %GOPATH%/src/github.com/ipfs/go-ipfs/cmd/ipfs
go install
```

Теперь у вас есть целый, собирающийся и готовый для разработки сервер CasperAPI.
### Linux-платформы
### Debian

Инструкция написана для Debian 9.3 Stretch с настроенным sudo, но может быть применена на ваш дистрибутив с незначительными изменениями. 

Выполнить следующие команды
```
sudo apt-get update
sudo apt-get install build-essentials
sudo apt-get install software-properties-common
sudo apt-get install cmake
sudo apt-get install libboost-all-dev
sudo apt-get install git
# Если у вас другое расположение $GOPATH, укажите другую директорию
sudo mkdir /go
# Права с пользователями и группами на свое усмотрение
sudo chown root:sudo /go
```

В /etc/sudoers в Defaults secure_path="" добавить следующие пути
```
/usr/local/go/bin
# Если у вас другое расположение $GOPATH, укажите другую директорию, вместо /go
/go/bin
```

**Установить golang 1.9 и выше.**
На момент написании статьи, **репозитории Debian 9.3 Stretch содержат версию 1.7**, поэтому необходимо установить Go самостоятельно, по [инструкции](https://golang.org/doc/install) с официального сайта

В /etc/profile и /root/.profile дописать следующие строки
```
# GO references
export PATH=$PATH:/usr/local/go/bin
# Если у вас другое расположение $GOPATH, укажите другую директорию
export GOPATH=/go
export PATH=$PATH:$GOPATH/bin
```

Скачиваем зависимости ipfs-go
```
sudo -E go get -u -d -v github.com/ipfs/go-ipfs
sudo -E rm -rf $GOPATH/src/github.com/ipfs/go-ipfs
sudo -E git clone -b dev_integration https://gitlab.com/casperDev/Casper-server.git $GOPATH/src/github.com/ipfs/go-ipfs
```
Устанавливаем другие внешние зависимости
```
sudo -E mkdir $GOPATH/src/gitlab.com/casperDev/
cd $GOPATH/src/gitlab.com/casperDev/
# В зависимости от ветки необходимо сделать git checkout или указать -b <branch>
sudo -E git clone https://gitlab.com/casperDev/contractsCasperApi.git
sudo -E git clone https://gitlab.com/casperDev/Casper-thrift.git
sudo -E git clone https://gitlab.com/casperDev/Casper-Golang_SC.git
sudo -E go get -u -d -v github.com/whyrusleeping/gx
sudo -E go get -u -d -v github.com/whyrusleeping/gx-go
sudo -E go get -u -d -v github.com/ethereum/go-ethereum
sudo -E go get -v git.apache.org/thrift.git/lib/go/thrift/...

# Installing gx and abigen from sources
cd $GOPATH/src/github.com/whyrusleeping/gx
sudo -E go install
cd $GOPATH/src/github.com/whyrusleeping/gx-go
sudo -E go install
cd $GOPATH/src/github.com/ethereum/go-ethereum/cmd/abigen
sudo -E go install
```
Устанавливаем зависимости ipfs-go через gx

```
cd $GOPATH/src/github.com/ipfs/go-ipfs
sudo -E gx --verbose install --global
```

Собираем последнюю версию Thrift протокола 
```
cd $GOPATH/src/gitlab.com/casperDev/Casper-thrift
sudo -E gen.sh
```
Устанавливаем [Solidity](http://solidity.readthedocs.io/en/develop/installing-solidity.html) любым доступным спосообом

Устанавливаем ipfs-go
```
cd $GOPATH/src/github.com/ipfs/go-ipfs
sudo -E make install
```

Генерируем последнюю версию контракта для вызовов
```
cd $GOPATH/src/gitlab.com/casperDev/contractsCasperApi
sudo -E abigen -sol contracts/Casper.sol -pkg casper > ../Casper-SC/casper/casper_sc.go
```
Проверьте, что в $GOPATH/src/gitlab.com/casperDev/Casper-SC/casper/casper_sc.go не написана ошибка. В файле должен быть валидный код на языке Go.

Финальный штрих
```
cd $GOPATH/src/github.com/ipfs/go-ipfs/cmd/ipfs
sudo -E go install
```

Бинарь лежит в $GOPATH/bin