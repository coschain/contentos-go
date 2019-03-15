#!/bin/bash

work_path=$(dirname $(readlink -f $0))
choice=`cat ${work_path}/set.txt`
case $choice in
    delete)
        cd /root/.coschain
        rm -rf *
        mkdir -p /root/.coschain/cosd
        cp /tmp/config.toml /root/.coschain/cosd/
        ;;
    reserve)
        ;;
    *)
        echo "Unknown choice, it should be delete or reserve, your choice is $choice"
        exit 1
        ;;
esac


#${work_path}/bin/cosd init
exec ${work_path}/bin/cosd start
