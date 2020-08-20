# Experimental Opera

This version of Opera is a testing/benchmarking platform, it is not safe to use.

## Building the source

Building `benchopera` requires both a Go (version 1.13 or later) and a C compiler. You can install
them using your favourite package manager. Once the dependencies are installed, run

```shell
go build -o ./build/benchopera ./cmd/benchopera
```
The build output is ```build/benchopera``` executable.

Do not clone the project into $GOPATH, due to the Go Modules. Instead, use any other location.

## Running `benchopera`

Going through all the possible command line flags is out of scope here,
but we've enumerated a few common parameter combos to get you up to speed quickly
on how you can run your own `benchopera` instance.

### Configuration

As an alternative to passing the numerous flags to the `benchopera` binary, you can also pass a
configuration file via:

```shell
$ benchopera --config /path/to/your_config.toml
```

To get an idea how the file should look like you can use the `dumpconfig` subcommand to
export your existing configuration:

```shell
$ benchopera --your-favourite-flags dumpconfig
```

## Dev

### Testing

Use the Go tool to run tests:
```shell
go test ./...
```

If everything goes well, it should output something along these lines:
```
?   	github.com/Fantom-foundation/benchopera/api	[no test files]
?   	github.com/Fantom-foundation/benchopera/benchopera	[no test files]
?   	github.com/Fantom-foundation/benchopera/benchopera/genesis	[no test files]
?   	github.com/Fantom-foundation/benchopera/cmd/benchopera	[no test files]
?   	github.com/Fantom-foundation/benchopera/cmd/benchopera/metrics	[no test files]
?   	github.com/Fantom-foundation/benchopera/debug	[no test files]
?   	github.com/Fantom-foundation/benchopera/eventcheck	[no test files]
?   	github.com/Fantom-foundation/benchopera/eventcheck/basiccheck	[no test files]
?   	github.com/Fantom-foundation/benchopera/eventcheck/heavycheck	[no test files]
?   	github.com/Fantom-foundation/benchopera/eventcheck/parentscheck	[no test files]
?   	github.com/Fantom-foundation/benchopera/gossip	[no test files]
?   	github.com/Fantom-foundation/benchopera/gossip/emitter	[no test files]
ok  	github.com/Fantom-foundation/benchopera/integration	10.078s
ok  	github.com/Fantom-foundation/benchopera/inter	(cached)
?   	github.com/Fantom-foundation/benchopera/logger	[no test files]
?   	github.com/Fantom-foundation/benchopera/metrics/prometheus	[no test files]
ok  	github.com/Fantom-foundation/benchopera/utils	(cached)
?   	github.com/Fantom-foundation/benchopera/utils/errlock	[no test files]
ok  	github.com/Fantom-foundation/benchopera/utils/fast	(cached)
ok  	github.com/Fantom-foundation/benchopera/utils/migration	(cached)
?   	github.com/Fantom-foundation/benchopera/utils/throughput	[no test files]
?   	github.com/Fantom-foundation/benchopera/version	[no test files]
```

### Operating a private network (fakenet)

Fakenet is a private network optimized for your private testing.
It'll generate a genesis containing N validators with equal stakes.
To launch a validator in this network, all you need to do is specify a validator ID you're willing to launch.

Pay attention that validator's private keys are deterministically generated in this network, so you must use it only for private testing.

Maintaining your own private network is more involved as a lot of configurations taken for
granted in the official networks need to be manually set up.

To run the fakenet with just one validator (which will work practically as a PoA blockchain), use:
```shell
$ benchopera --fakenet 1/1
```

To run the fakenet with 5 validators, run the command for each validator:
```shell
$ benchopera --fakenet 1/5 # first node, use 2/5 for second node
```

If you have to launch a non-validator node in fakenet, use 0 as ID:
```shell
$ benchopera --fakenet 0/5
```

After that, you have to connect your nodes. Either connect them statically or specify a bootnode:
```shell
$ benchopera --fakenet 1/5 --bootnodes "enode://verylonghex@1.2.3.4:5050"
```

### Testing event payload
```shell
$ benchopera --bps=600000 --txpayload=100000
```
This combination of flags means "emit no more than 600000 bytes per second, place 100000 bytes of payload into each event"

### Running the demo

For the testing purposes, the full demo may be launched using:
```shell
cd demo/
./start.sh # start the fakenet instances
./stop.sh # stop the demo
```
