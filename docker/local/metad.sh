#!/bin/sh

set -e

Metaplayerone_CHAIN_BIN=./metachain
CHAIN_CUSTOM_OPTIONS=$(tr "\n" " " << EOL
--block-gas-limit 10000000
--epoch-size 10
--chain-id 51001
--name metachain-docker
--premine 0x0000000000000000000000000000000000000000
--premine 0x228466F2C715CbEC05dEAbfAc040ce3619d7CF0B:0xD3C21BCECCEDA1000000
--premine 0xca48694ebcB2548dF5030372BE4dAad694ef174e:0xD3C21BCECCEDA1000000
--burn-contract 0:0x0000000000000000000000000000000000000000
EOL
)

case "$1" in
   "init")
      case "$2" in 
          "metabft")
              if [ -f "$GENESIS_PATH" ]; then
                  echo "Secrets have already been generated."
              else
                  echo "Generating METABFT secrets..."
                  secrets=$("$Metaplayerone_CHAIN_BIN" secrets init --insecure --num 4 --data-dir /data/data- --json)
                  echo "Secrets have been successfully generated"

                  rm -f /data/genesis.json

                  echo "Generating METABFT Genesis file..."
                  "$Metaplayerone_CHAIN_BIN" genesis $CHAIN_CUSTOM_OPTIONS \
                    --dir /data/genesis.json \
                    --consensus metabft \
                    --metabft-validators-prefix-path data- \
                    --bootnode "/dns4/node-1/tcp/1478/p2p/$(echo "$secrets" | jq -r '.[0] | .node_id')" \
                    --bootnode "/dns4/node-2/tcp/1478/p2p/$(echo "$secrets" | jq -r '.[1] | .node_id')" \
                    --bootnode "/dns4/node-3/tcp/1478/p2p/$(echo "$secrets" | jq -r '.[2] | .node_id')" \
                    --bootnode "/dns4/node-4/tcp/1478/p2p/$(echo "$secrets" | jq -r '.[3] | .node_id')"
              fi
              ;;
          "mbft")
              echo "Generating mbft secrets..."
              secrets=$("$Metaplayerone_CHAIN_BIN" mbft-secrets init --insecure --num 4 --data-dir /data/data- --json)
              echo "Secrets have been successfully generated"

              rm -f /data/genesis.json

              proxyContractsAdmin=0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed

              echo "Generating MBFT genesis file..."
              "$Metaplayerone_CHAIN_BIN" genesis $CHAIN_CUSTOM_OPTIONS \
                --dir /data/genesis.json \
                --consensus mbft \
                --validators-path /data \
                --validators-prefix data- \
                --reward-wallet 0xDEADBEEF:1000000 \
                --native-token-config "Metaunit:MEU:18:true:$(echo "$secrets" | jq -r '.[0] | .address')" \
                --proxy-contracts-admin ${proxyContractsAdmin} \
                --bootnode "/dns4/node-1/tcp/1478/p2p/$(echo "$secrets" | jq -r '.[0] | .node_id')" \
                --bootnode "/dns4/node-2/tcp/1478/p2p/$(echo "$secrets" | jq -r '.[1] | .node_id')" \
                --bootnode "/dns4/node-3/tcp/1478/p2p/$(echo "$secrets" | jq -r '.[2] | .node_id')" \
                --bootnode "/dns4/node-4/tcp/1478/p2p/$(echo "$secrets" | jq -r '.[3] | .node_id')"

              echo "Deploying stake manager..."
              "$Metaplayerone_CHAIN_BIN" mbft stake-manager-deploy \
                --jsonrpc http://rootchain:8545 \
                --genesis /data/genesis.json \
                --proxy-contracts-admin ${proxyContractsAdmin} \
                --test

              stakeManagerAddr=$(cat /data/genesis.json | jq -r '.params.engine.mbft.bridge.stakeManagerAddr')
              stakeToken=$(cat /data/genesis.json | jq -r '.params.engine.mbft.bridge.stakeTokenAddr')

              "$Metaplayerone_CHAIN_BIN" rootchain deploy \
                --stake-manager ${stakeManagerAddr} \
                --stake-token ${stakeToken} \
                --json-rpc http://rootchain:8545 \
                --genesis /data/genesis.json \
                --proxy-contracts-admin ${proxyContractsAdmin} \
                --test

              customSupernetManagerAddr=$(cat /data/genesis.json | jq -r '.params.engine.mbft.bridge.customSupernetManagerAddr')
              supernetID=$(cat /data/genesis.json | jq -r '.params.engine.mbft.supernetID')
              addresses="$(echo "$secrets" | jq -r '.[0] | .address'),$(echo "$secrets" | jq -r '.[1] | .address'),$(echo "$secrets" | jq -r '.[2] | .address'),$(echo "$secrets" | jq -r '.[3] | .address')"

              "$Metaplayerone_CHAIN_BIN" rootchain fund \
                --json-rpc http://rootchain:8545 \
                --stake-token ${stakeToken} \
                --mint \
                --addresses ${addresses} \
                --amounts 1000000000000000000000000,1000000000000000000000000,1000000000000000000000000,1000000000000000000000000

              "$Metaplayerone_CHAIN_BIN" mbft whitelist-validators \
                --addresses ${addresses} \
                --supernet-manager ${customSupernetManagerAddr} \
                --private-key aa75e9a7d427efc732f8e4f1a5b7646adcc61fd5bae40f80d13c8419c9f43d6d \
                --jsonrpc http://rootchain:8545

              counter=1
              while [ $counter -le 4 ]; do
                echo "Registering validator: ${counter}"

                "$Metaplayerone_CHAIN_BIN" mbft register-validator \
                  --supernet-manager ${customSupernetManagerAddr} \
                  --data-dir /data/data-${counter} \
                  --jsonrpc http://rootchain:8545

                "$Metaplayerone_CHAIN_BIN" mbft stake \
                  --data-dir /data/data-${counter} \
                  --amount 1000000000000000000000000 \
                  --supernet-id ${supernetID} \
                  --stake-manager ${stakeManagerAddr} \
                  --stake-token ${stakeToken} \
                  --jsonrpc http://rootchain:8545

                counter=$((counter + 1))
              done

              "$Metaplayerone_CHAIN_BIN" mbft supernet \
                --private-key aa75e9a7d427efc732f8e4f1a5b7646adcc61fd5bae40f80d13c8419c9f43d6d \
                --supernet-manager ${customSupernetManagerAddr} \
                --stake-manager ${stakeManagerAddr} \
                --finalize-genesis-set \
                --enable-staking \
                --genesis /data/genesis.json \
                --jsonrpc http://rootchain:8545
              ;;
      esac
      ;;
   *)
      echo "Executing metachain..."
      exec "$Metaplayerone_CHAIN_BIN" "$@"
      ;;
esac
