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
<br>![image](https://github.com/ygqbasic/cloud/blob/master/image/1.png)<br>
