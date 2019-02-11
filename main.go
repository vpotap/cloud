package main

import (
	_ "cloud/routers"

	"github.com/astaxie/beego"
	_ "github.com/astaxie/beego/session/redis"
	_ "github.com/go-sql-driver/mysql"
)

//https://ant.design/docs/resource/download-cn
//https://beego.me/docs/mvc/controller/session.md
//https://www.kubernetes.org.cn/configmap
//https://kubernetes.io/docs/concepts/api-extension/apiserver-aggregation/
//http://blog.csdn.net/dream_broken/article/details/53130515
// https://kubernetes.io/docs/reference/
//http://developer.alauda.cn/usermanual/features/servicecreate.html
// http://yoyolive.com/2017/03/09/Kubernetes-Deploy-GlusterFS/
// http://www.360doc.com/content/18/0105/18/17050303_719342191.shtml glusterfs http://yoyolive.com/2017/03/09/Kubernetes-Deploy-GlusterFS/
// http://docs.tenxcloud.com/guide/coderepos
// 2019-01-26 15:11

func main() {
	//beego.ErrorController(&index.ErrorController{})
	beego.BConfig.WebConfig.Session.SessionProvider = "redis"
	beego.BConfig.WebConfig.Session.SessionProviderConfig = beego.AppConfig.String("redis")
	//go perm.UpdateResource()
	//go tty.TtyStart()

	//go auth_server.StartRegistryAuthServer()
	//go crontab.CronStart()
	beego.Run()
}
