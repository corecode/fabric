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

peerconf=""

for n in $(seq $count)
do
    cat > template$n.cfg <<EOF
expiration_days = -1
serial = $(date +"%N")
signing_key
encryption_key
EOF
    certtool --generate-self-signed --load-privkey key.pem --outfile cert$n.pem --template template$n.cfg 2>/dev/null
    certtool -i --infile cert$n.pem --outder --outfile data/config.peers.:$((6100+$n)) 2>/dev/null

    peerconf=$(cat <<EOF
${peerconf}${peerconf:+,}
{
  "address": ":$((6100+$n))",
  "cert": "cert${n}.pem"
}
EOF
)
done

cat > config.json <<EOF
{
  "consensus": {
    "pbft_config": {
      "f": $fail,
      "k": 2,
      "l_multiplier": 2,
      "request_timeout": 2.0,
      "viewchange_resend_timeout": 2.0,
      "viewchange_timeout": 2.0
    }
  },
  "peers": [${peerconf}]
}
EOF

for n in $(seq $count)
do
    consensus-peer -data-dir data$n -init config.json
    cat > run-$n.sh <<EOF
#!/bin/sh
consensus-peer -addr :$((6100+$n)) -cert cert$n.pem -key key.pem -data-dir data$n "\$@"
EOF
    chmod +x run-$n.sh
done
