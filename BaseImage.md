# GItLab CI 基础镜像列表

特别注意: 制作.gitlab.ci 必须要用以下基础镜像 然后再用镜像分层,构建出自己需要的镜像，基础镜像里面已经存放了GitLab clone 免密登录证书, 
禁止私自修改基础镜像

```$xslt
registry.cn-hangzhou.aliyuncs.com/guanghe/cigolang:latest
```

```$xslt
registry.cn-hangzhou.aliyuncs.com/guanghe/cidocker:latest
```

```$xslt
registry.cn-hangzhou.aliyuncs.com/guanghe/cinode:latest
```