#!/bin/sh

set -e

dest=$1
count=$2

if test -z "$count"
then
    echo "usage: local-deploy.sh <destdir> <nreplica>" >&1
    exit 1
fi


fail=$((($count - 1)/3))

mkdir $dest
cd $dest

certtool --generate-privkey --outfile key.pem 2>/dev/null
mkdir data
cat > template.cfg <<EOF
expiration_days = -1
signing_key
encryption_key
EOF

for n in $(seq $count)
do
    certtool --generate-self-signed --load-privkey key.pem --outfile cert$n.pem --template template.cfg 2>/dev/null
    certtool -i --infile cert$n.pem --outder --outfile data/config.peers.:$((6100+$n)) 2>/dev/null
done

for n in $(seq $count)
do
    cp -R data data$n
    cat > run-$n.sh <<EOF
#!/bin/sh
export CORE_PBFT_GENERAL_N=$count
export CORE_PBFT_GENERAL_F=$fail
consensus-peer -addr :$((6100+$n)) -cert cert$n.pem -key key.pem -data-dir data$n
EOF
    chmod +x run-$n.sh
done
