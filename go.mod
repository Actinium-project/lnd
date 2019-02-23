module github.com/Actinium-project/lnd

require (
	git.schwanenlied.me/yawning/bsaes.git v0.0.0-20190222205624-e466e3c35f68 // indirect
	github.com/Actinium-project/acmd v0.0.16
	github.com/Actinium-project/acmutil v0.0.9
	github.com/Actinium-project/acmwallet v0.0.5
	github.com/Actinium-project/actrino v0.0.2
	github.com/Actinium-project/lightning-onion v0.0.2
	github.com/NebulousLabs/fastrand v0.0.0-20181203155948-6fb6489aac4e // indirect
	github.com/NebulousLabs/go-upnp v0.0.0-20180202185039-29b680b06c82
	github.com/Yawning/aez v0.0.0-20180114000226-4dad034d9db2
	github.com/btcsuite/btcd v0.0.0-20190213025234-306aecffea32
	github.com/btcsuite/btclog v0.0.0-20170628155309-84c8d2346e9f
	github.com/btcsuite/fastsha256 v0.0.0-20160815193821-637e65642941
	github.com/coreos/bbolt v1.3.2
	github.com/davecgh/go-spew v1.1.1
	github.com/go-errors/errors v1.0.1
	github.com/golang/protobuf v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v0.0.0-20170724004829-f2862b476edc
	github.com/jackpal/gateway v1.0.4
	github.com/jackpal/go-nat-pmp v0.0.0-20170405195558-28a68d0c24ad
	github.com/jessevdk/go-flags v1.4.0
	github.com/jrick/logrotate v1.0.0
	github.com/juju/loggo v0.0.0-20190212223446-d976af380377 // indirect
	github.com/kkdai/bstream v0.0.0-20181106074824-b3251f7901ec
	github.com/lightninglabs/neutrino v0.0.0-20190219013218-1a80fd3d0e92 // indirect
	github.com/lightningnetwork/lnd v0.0.2
	github.com/lightningnetwork/lnd/queue v1.0.0
	github.com/miekg/dns v0.0.0-20171125082028-79bfde677fa8
	github.com/roasbeef/btcd v0.0.0-20180418012700-a03db407e40d // indirect
	github.com/roasbeef/btcutil v0.0.0-20180406014609-dfb640c57141 // indirect
	github.com/roasbeef/btcwallet v0.0.0-20180426223453-30affec83c18 // indirect
	github.com/rogpeppe/fastuuid v1.0.0 // indirect
	github.com/tv42/zbase32 v0.0.0-20160707012821-501572607d02
	github.com/urfave/cli v1.18.0
	golang.org/x/crypto v0.0.0-20190211182817-74369b46fc67
	golang.org/x/net v0.0.0-20190206173232-65e2d4e15006
	golang.org/x/time v0.0.0-20180412165947-fbb02b2291d2
	google.golang.org/genproto v0.0.0-20190201180003-4b09977fb922
	google.golang.org/grpc v1.18.0
	gopkg.in/macaroon-bakery.v2 v2.0.1
	gopkg.in/macaroon.v2 v2.0.0
)

replace github.com/lightningnetwork/lnd/ticker => ./ticker

replace github.com/lightningnetwork/lnd/queue => ./queue
