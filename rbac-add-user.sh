#!/bin/bash

if [[  -n  $1 ]] ; then
        USERNAME=$1
else
        echo "请传入参数,参数为需要创建的用户名"
        echo "Use :    k8s-add-user.sh    username cluster-name"
        exit 1
fi
cluster_name=$2

kubectl  create sa  $USERNAME
USERSECRET=$(kubectl  get  secret   | awk  '/'$USERNAME'/{print $1}')
TOKEN=$(kubectl  get     secret  $USERSECRET  --template={{.data.token}}  | base64 -d)
kubectl  create   clusterrolebinding $USERNAME-crb --clusterrole=cluster-admin --serviceaccount=default:$USERNAME
kubectl  config   set-cluster $cluster_name  --server="https://10.8.8.207:6443" --certificate-authority=/root/.minikube/ca.crt  --embed-certs=true --kubeconfig=/tmp/$USERNAME.conf
kubectl  config   set-credentials $USERNAME-$cluster_name  --kubeconfig=/tmp/$USERNAME.conf  --token=$TOKEN
kubectl  config   set-context $cluster_name   --cluster=$cluster_name  --user=$USERNAME-$cluster_name --kubeconfig=/tmp/$USERNAME.conf