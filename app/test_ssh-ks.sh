#!/usr/bin/env bash


. ./libtest.sh
DBG_SHOW=1
# Debug-level for server
DBG_SRV=1
# Debug-level for client
DBG_CLIENT=1

main(){
    startTest
    build
    test Build
    test Cothorityd
    test ClientSetup
    test ClientAdd
    test ServerSetup
    stopTest
}

testServerSetup(){
    coClient
    coClientSetup
    testOK runSrv 1 setup group.toml $ID
    testGrep client_1 runSrv 1 list
    testNGrep client_2 runSrv 1 list

    # Now add a second client and verify its in the list
    testOK runCl 1 confirm
    testOK runSrv 1 update
    testGrep client_1 runSrv 1 list
    testGrep client_2 runSrv 1 list

    testOut "Adding identity 'user'"
    coClientSetup user
    testOK runSrv 1 setup group.toml $ID
    testGrep client_1 runSrv 1 list
    testGrep client_2 runSrv 1 list
    testGrep user_1 runSrv 1 list
    testNGrep user_2 runSrv 1 list
    testOK runCl 1 confirm
    testOK runSrv 1 update
    testGrep user_2 runSrv 1 list
}

testClientAdd(){
    coClient
    # Setting up first client
    testOK runCl 1 setup -n client_1 group.toml
    testOK runGrep Identity-ID: runCl 1 list
    ID=$( echo $GRP | sed -e "s/.*: //" )

    # Adding second client
    # Test we can't add the same account-name twice
    testFail runCl 2 setup -n client_1 -a $ID group.toml
    # Test the ID is checked
    testFail runCl 2 setup -n client_2 -a 11$ID group.toml
    testOK runCl 2 setup -n client_2 -a $ID group.toml
    testGrep client_2 runCl 1 listNew

    # Confirm second client
    testOK runCl 1 confirm
    testNGrep "Proposed config" runCl 1 list
    testOK runCl 2 update
    testNGrep "Proposed config" runCl 2 list
    testGrep client_2 runCl 2 list
}

testClientSetup(){
    coClient
    testOK runCl 1 setup group.toml
    testFile cl1/config.bin
}

coClientSetup(){
    DBG_OLD=$DBG_SHOW
    DBG_SHOW=0
    CLIENT=${1:-client}
    runCl 1 setup -n ${CLIENT}_1 group.toml
    runGrep Identity-ID: runCl 1 list
    ID=$( echo $GRP | sed -e "s/.*: //" )
    runCl 2 setup -n ${CLIENT}_2 -a $ID group.toml
    runCl 1 update
    DBG_SHOW=$DBG_OLD
}

coClient(){
    DBG_OLD=$DBG_SHOW
    DBG_SHOW=0
    runCoCfg 1
    runCoCfg 2
    runCoCfg 3
    runCoBG 1
    runCoBG 2
    sleep 1
    cp co1/group.toml .
    tail -n 4 co2/group.toml >> group.toml
    DBG_SHOW=$DBG_OLD
}

testCothorityd(){
    runCoCfg 1
    runCoCfg 2
    runCoCfg 3
    runCoBG 1
    runCoBG 2
    sleep 1
    cp co1/group.toml .
    tail -n 4 co2/group.toml >> group.toml
    testOK runCl 1 check group.toml
    tail -n 4 co3/group.toml >> group.toml
    testFail runCl 1 check group.toml
}

testBuild(){
    testOK dbgRun ./cothorityd --help
    testOK dbgRun ./ssh-kss --help
    testOK dbgRun ./ssh-ksc -c cl1 -cs cl1 --help
}

runCl(){
    D=cl$1
    shift
    dbgRun ./ssh-ksc -d $DBG_CLIENT -c $D --cs $D $@
}

runSrv(){
    nb=$1
    shift
    dbgRun ./ssh-kss -d $DBG_SRV -c srv$nb -cs srv$nb $@
}

runCoCfg(){
    echo -e "127.0.0.1:200$1\nco$1\n\n" | dbgRun runCo $1 setup
}

runCoBG(){
    nb=$1
    shift
    ( ./cothorityd -d $DBG_SRV -c co$nb/config.toml $@ 2>&1 > /dev/null & )
}

runCo(){
    nb=$1
    shift
    dbgRun ./cothorityd -d $DBG_SRV -c co$nb/config.toml $@
}

build(){
    BUILDDIR=$(pwd)
    if [ "$STATICDIR" ]; then
        DIR=$STATICDIR
    else
        DIR=$(mktemp -d)
    fi
    mkdir -p $DIR
    cd $DIR
    echo "Building in $DIR"
    for app in cothorityd ssh-ksc ssh-kss; do
        if [ ! -e $app -o "$BUILD" ]; then
            if ! go build -o $app $BUILDDIR/$app/*.go; then
                fail "Couldn't build $app"
            fi
        fi
    done
    echo "Creating keys"
    for n in $(seq $NBR); do
        srv=srv$n
        if [ -d $srv ]; then
            rm -f $srv/*bin
        else
            mkdir $srv
            ssh-keygen -t rsa -b 4096 -N "" -f $srv/ssh_host_rsa_key > /dev/null
        fi

        cl=cl$n
        if [ -d $cl ]; then
            rm -f $cl/*bin
        else
            mkdir $cl
            ssh-keygen -t rsa -b 4096 -N "" -f $cl/id_rsa > /dev/null
        fi

        co=co$n
        rm -f $co/*
        mkdir -p $co
    done
}

if [ "$1" -a "$STATICDIR" ]; then
    rm -f $STATICDIR/{cothorityd,ssh-ks{c,s}}
fi

main