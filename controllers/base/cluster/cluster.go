package cluster

import (
	"cloud/cache"
	"cloud/controllers/base/hosts"
	"cloud/k8s"
	"cloud/models/cluster"
	hosts2 "cloud/models/hosts"
	"cloud/sql"
	"cloud/util"
	"strings"

	"github.com/astaxie/beego"
)

type ClusterController struct {
	beego.Controller
}

// 集群管理入口页面
// @router /base/cluster/index [get]
func (this *ClusterController) List() {
	this.TplName = "base/cluster/index.html"
}

// 集群管理入口页面
// @router /base/cluster/image/:id:int [get]
func (this *ClusterController) Images() {
	this.Data["hostId"] = this.Ctx.Input.Param(":id")
	this.TplName = "base/cluster/img.html"
}

// 节点报表入口页面
// @router /base/cluster/report/:id:int [get]
func (this *ClusterController) Report() {
	this.Data["hostId"] = this.Ctx.Input.Param(":id")
	this.TplName = "base/cluster/report.html"
}

// 添加集群
// @router /base/cluster/add [get]
func (this *ClusterController) Add() {
	id, _ := this.GetInt("ClusterId")
	update := cluster.CloudCluster{}
	update.NetworkCart = "6443"
	if id != 0 {
		q := sql.SearchSql(update, cluster.SelectCloudCluster, sql.GetSearchMap("ClusterId", *this.Ctx))
		sql.Raw(q).QueryRow(&update)
		h, p := k8s.GetMasterIp(update.ClusterName)
		update.ApiAddress = h
		update.NetworkCart = p
	}

	this.Data["data"] = update
	this.TplName = "base/cluster/add.html"
}

// @router /base/cluster/detail/:hi(.*) [get]
func (this *ClusterController) DetailPage() {
	name := this.Ctx.Input.Param(":hi")
	if len(name) < 1 {
		this.Redirect("/base/cluster/list", 302)
		return
	}

	detail := GetClusterDetailData(name)
	if detail.ClusterId < 1 {
		this.Redirect("/base/cluster/list", 302)
		return
	}
	this.Data["data"] = detail
	this.TplName = "base/cluster/detail.html"
}

// 保存集群，初始化集群节点IP
// json
// @router /api/cluster [post]
func (this *ClusterController) Save() {
	d := cluster.CloudCluster{}
	err := this.ParseForm(&d)
	if err != nil {
		this.Ctx.WriteString("参数错误" + err.Error())
		return
	}
	util.SetPublicData(d, getUsername(this), &d)
	q := sql.InsertSql(d, cluster.InsertCloudCluster)
	if d.ClusterId > 0 {
		searchMap := sql.SearchMap{}
		searchMap.Put("ClusterId", d.ClusterId)
		q = sql.UpdateSql(d, cluster.UpdateCloudCluster, searchMap, "CreateTime,CreateUser")
	}

	_, err = sql.Raw(q).Exec()
	// 删除已有的主机节点
	searchMap := sql.SearchMap{}
	searchMap.Put("ClusterName", d.ClusterName)
	sql.Exec(sql.DeleteSql(hosts2.DeleteCloudClusterHosts, searchMap))
	cache.MasterCache.Delete(d.ClusterName)

	// 插入集群节点数据
	h := hosts2.CloudClusterHosts{}
	h.HostType = "master"
	h.HostIp = d.ApiAddress
	h.ApiPort = d.NetworkCart
	h.CreateTime = util.GetDate()
	h.CreateUser = getUsername(this)
	h.LastModifyTime = h.CreateTime
	h.LastModifyUser = h.CreateUser
	h.ClusterName = d.ClusterName
	i := sql.InsertSql(h, hosts2.InsertCloudClusterHosts)
	sql.Raw(i).Exec()
	CacheClusterData()
	data, msg := util.SaveResponse(err, "名称已经被使用")
	util.SaveOperLog(this.GetSession("username"), *this.Ctx, "保存集群操作 "+msg, d.ClusterName)
	setClusterJson(this, data)
}

// json响应
// 集群数据,直返回,集群名称和id的数据
// @router /api/cluster/name [get]
func (this *ClusterController) ClusterName() {
	setClusterJson(this, GetClusterName())
}

// json 响应
// 集群数据获取
// @router /api/cluster [get]
func (this *ClusterController) ClusterData() {
	searchMap := sql.SearchMap{}
	id := this.Ctx.Input.Param(":id")
	key := this.GetString("key")
	if id != "" {
		searchMap.Put("ClusterId", id)
	}

	searchSql := sql.SearchSql(
		cluster.CloudCluster{},
		cluster.SelectCloudCluster,
		searchMap)

	if key != "" && id == "" {
		pkey := sql.Replace(key)
		searchSql += strings.Replace(cluster.SelectCloudClusterWhere, "?", pkey, -1)
	}
	data := make([]k8s.ClusterStatus, 0)
	sql.Raw(searchSql).QueryRows(&data)
	result := make([]k8s.ClusterStatus, 0)
	for _, v := range data {
		r := cache.ClusterCache.Get("data" + v.ClusterName)
		v1 := k8s.ClusterStatus{}
		status := util.RedisObj2Obj(r, &v1)
		if status {
			result = append(result, v1)
		} else {
			result = append(result, v)
			CacheClusterData()
		}
	}
	var r = util.ResponseMap(result, len(result), 1)
	setClusterJson(this, r)
}

// @router /api/cluster/nodes [get]
func (this *ClusterController) NodesData() {
	clusterName := this.GetString("clusterName")
	var check bool = true
	c, err := k8s.GetClient(clusterName)
	if err != nil {
		check = false
	}
	if !check {
		setClusterJson(this, k8s.NodeIp{})
		return
	}
	this.Data["json"] = k8s.GetNodesIp(c)
	this.ServeJSON(false)
}

// @router /api/cluster/delete [*]
func (this *ClusterController) Delete() {
	searchMap := sql.SearchMap{}
	id := this.Ctx.Input.Param(":id")
	searchMap.Put("ClusterId", id)
	cloudCluster := cluster.CloudCluster{}

	q := sql.SearchSql(cloudCluster, cluster.SelectCloudCluster, searchMap)
	sql.Raw(q).QueryRow(&cloudCluster)

	size := len(hosts.GetClusterHosts(cloudCluster.ClusterName))
	if size > 0 {
		msg := "删除失败: 该集群还有节点没有清理,不能删除"
		r := util.ApiResponse(false, msg)
		util.SaveOperLog(
			this.GetSession("username"),
			*this.Ctx, "删除集群 "+msg,
			cloudCluster.ClusterName)

		setClusterJson(this, r)
		return
	}

	q = sql.DeleteSql(cluster.DeleteCloudCluster, searchMap)
	r, err := sql.Raw(q).Exec()
	data := util.DeleteResponse(
		err,
		*this.Ctx,
		"删除集群"+cloudCluster.ClusterName,
		this.GetSession("username"),
		cloudCluster.ClusterName,
		r)
	setClusterJson(this, data)
}

// json响应
// v1 集群数据,直返回,集群名称和id的数据
// @router /api/v1/cluster/name [get]
func (this *ClusterController) GetClusterName() {
	var r = util.RestApiResponse(200, GetClusterName())
	setClusterJson(this, r)
}

// json 响应
// v1 集群数据获取
// @router /api/v1/clusters [get]
func (this *ClusterController) Clusters() {
	searchMap := sql.SearchMap{}
	id := this.Ctx.Input.Param(":id")
	key := this.GetString("key")
	if id != "" {
		searchMap.Put("ClusterId", id)
	}

	searchSql := sql.SearchSql(
		cluster.CloudCluster{},
		cluster.SelectCloudCluster,
		searchMap)

	if key != "" && id == "" {
		pkey := sql.Replace(key)
		searchSql += strings.Replace(cluster.SelectCloudClusterWhere, "?", pkey, -1)
	}
	data := make([]k8s.ClusterStatus, 0)
	sql.Raw(searchSql).QueryRows(&data)
	result := make([]k8s.ClusterStatus, 0)
	for _, v := range data {
		r := cache.ClusterCache.Get("data" + v.ClusterName)
		v1 := k8s.ClusterStatus{}
		status := util.RedisObj2Obj(r, &v1)
		if status {
			result = append(result, v1)
		} else {
			result = append(result, v)
			CacheClusterData()
		}
	}
	var r = util.RestApiResponse(200, result)

	setClusterJson(this, r)
}

// json 响应
// v1 获取集群详情
// @router /api/v1/cluster/detail/:hi(.*) [get]
func (this *ClusterController) ClusterDetail() {
	name := this.Ctx.Input.Param(":hi")
	var r map[string]interface{}
	if len(name) < 1 {
		r = util.RestApiResponse(50001, "参数格式不正确")
	}

	detail := GetClusterDetailData(name)
	if detail.ClusterId < 1 {
		r = util.RestApiResponse(50001, "要查询的集群信息不存在")
	} else {
		r = util.RestApiResponse(200, detail)
	}

	setClusterJson(this, r)
}

// @router /api/v1/cluster/nodes [get]
func (this *ClusterController) ClusterNodes() {
	clusterName := this.GetString("clusterName")
	var check bool = true
	var r map[string]interface{}
	c, err := k8s.GetClient(clusterName)
	if err != nil {
		check = false
	}
	if !check {
		r = util.RestApiResponse(50001, "获取Kubernetes 的Client异常")
	} else {
		r = util.RestApiResponse(200, k8s.GetNodesIp(c))
	}
	setClusterJson(this, r)
}
