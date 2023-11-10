#/bin/bash
NS=keti-system
NAME=$(kubectl get pod -n $NS -o wide | grep -E 'metric-collector-c1' | grep -E 'master'| awk '{print $1}')

echo $NAME

kubectl logs -f -n $NS $NAME

