# DevOps  k8s命令行自动部署工具 使用指南
 
##DevOps宗旨
1 GitLab-CI 持续集成，k8s自动部署,滚动更新
2 自动域名发现
3 服务配置和镜像完全分离，配置由k8s configMap管理，底层基于k8s etcd 自动发现，更改配置时，无需
  重打镜像，
  
##注意事项:服务配置和镜像完全分离，配置由k8s configMap管理，服务监听端口必须写在配置文件里，配置交由configmap管理，禁止将配置文件
和二进制文件一起打包成到 docker image 中。服务监听端口不得写死，得写在配置config 中


## 创建一个namespace 
dep ns test
	
##部署后端服务
1先创建configMap, 将 config.yaml 放到 congfig.tpl.yml data 下面
 dep map test activity configtpl.yml

2 创建deployment,创建deployment 前确保同名image 已经推到registry.cn-hangzhou.aliyuncs.com
dep dep test activity deploytpl.yml

3 创建service

dep svc test activity 

## 部署一个无配置的服务
dep front test go-hello  registry.cn-hangzhou.aliyuncs.com/guanghe/golang:latest

### 常用命令
查看pod 运行情况
kubectl logs pod -n test

查看configMap
kubectl get cm go-hello -n test -o yaml

查看deployment

滚动更新

修改配置


# K8sDev
