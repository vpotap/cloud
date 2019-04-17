# Cloud

云原生发布运维平台

[![GitHub license](https://img.shields.io/github/license/ygqbasic/cloud.svg?style=flat)](https://github.com/ygqbasic/cloud/blob/master/LICENSE) 
![GitHub repo-size](https://img.shields.io/github/repo-size/ygqbasic/cloud.svg?style=flat)
![GitHub release-date](https://img.shields.io/github/release-date-pre/ygqbasic/cloud.svg?style=flat)
![GitHub release-pre](https://img.shields.io/github/release-pre/ygqbasic/cloud.svg?style=flat)
![GitHub contributors](https://img.shields.io/github/contributors/ygqbasic/cloud.svg?style=flat)
## 开始
### 安装Go环境
#### 安装govendor模块
```
go get -u -v github.com/kardianos/govendor
```
#### 编译源码
```
cd $GoPath/src/cloud
go build
```

## 发布部署
### linux
```
bee pack -be GOOS=linux
```

### windows
```
bee pack -be GOOS=windows
```

### unzip
```
mkdir cloud 
chmod -R 777 cloud
tar -xzvf cloud.tar.gz -C cloud
```
<br>![image](https://github.com/ygqbasic/ygqbasic.github.io/blob/master/styles/images/cloud/clusterinfo.png?raw=true)<br>


## Kubernetes 相关
### 自签名证书访问集群
- 证书在 /etc/kubernetes/ssl 下
- 执行
```sh
openssl pkcs12 -export -out admin.pfx -inkey admin-key.pem -in admin.pem -certfile ca.pem
```
- 拷贝生成的admin.pfx 证书文件到客户端机器，并导入证书
- hosts 文件中添加
```sh
ip kubernetes
ip kubernetes.default
ip kubernetes.default.svc
ip kubernetes.default.svc.cluster
ip kubernetes.default.svc.cluster.local
```
- 可以使用上面的域名访问kubernetes集群