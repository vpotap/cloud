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
<br>![image](https://github.com/ygqbasic/zcloud/blob/master/image/1.png)<br>
