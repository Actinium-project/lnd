module github.com/Actinium-project/lnd

require (
	git.schwanenlied.me/yawning/bsaes.git v0.0.0-20190222205624-e466e3c35f68 // indirect
	github.com/Actinium-project/acmd v0.0.17
	github.com/Actinium-project/acmutil v0.0.11
	github.com/Actinium-project/acmwallet v0.0.9-0.20190404102556-a382059d5b6c
	github.com/Actinium-project/actrino v0.0.5
	github.com/Actinium-project/lightning-onion v0.0.4
	github.com/Actinium-project/lnd/queue v1.0.0
	github.com/Actinium-project/lnd/ticker v1.0.0
	github.com/NebulousLabs/fastrand v0.0.0-20181203155948-6fb6489aac4e // indirect
	github.com/NebulousLabs/go-upnp v0.0.0-20180202185039-29b680b06c82
	github.com/Yawning/aez v0.0.0-20180114000226-4dad034d9db2
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
	github.com/juju/clock v0.0.0-20190205081909-9c5c9712527c // indirect
	github.com/juju/errors v0.0.0-20190207033735-e65537c515d7 // indirect
	github.com/juju/loggo v0.0.0-20190212223446-d976af380377 // indirect
	github.com/juju/retry v0.0.0-20180821225755-9058e192b216 // indirect
	github.com/juju/testing v0.0.0-20180920084828-472a3e8b2073 // indirect
	github.com/juju/utils v0.0.0-20180820210520-bf9cc5bdd62d // indirect
	github.com/juju/version v0.0.0-20180108022336-b64dbd566305 // indirect
	github.com/kkdai/bstream v0.0.0-20181106074824-b3251f7901ec
	github.com/miekg/dns v0.0.0-20171125082028-79bfde677fa8
	github.com/rogpeppe/fastuuid v1.0.0 // indirect
	github.com/tv42/zbase32 v0.0.0-20160707012821-501572607d02
	github.com/urfave/cli v1.18.0
	golang.org/x/crypto v0.0.0-20190211182817-74369b46fc67
	golang.org/x/net v0.0.0-20190206173232-65e2d4e15006
	golang.org/x/time v0.0.0-20180412165947-fbb02b2291d2
	google.golang.org/genproto v0.0.0-20190201180003-4b09977fb922
	google.golang.org/grpc v1.18.0
	gopkg.in/errgo.v1 v1.0.0 // indirect
	gopkg.in/macaroon-bakery.v2 v2.0.1
	gopkg.in/macaroon.v2 v2.0.0
	gopkg.in/mgo.v2 v2.0.0-20180705113604-9856a29383ce // indirect
)

replace github.com/Actinium-project/lnd/ticker => ./ticker

replace github.com/Actinium-project/lnd/queue => ./queue
