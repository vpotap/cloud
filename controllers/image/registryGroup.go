package registry

import (
	"cloud/controllers/base/cluster"
	"cloud/k8s"
	"cloud/models/registry"
	"cloud/sql"
	"cloud/util"
	"database/sql/driver"
	"strings"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/ygqbasic/go-harbor"
	"golang.org/x/crypto/openpgp/errors"

	"cloud/cache"
	"cloud/controllers/base/quota"
	"cloud/userperm"
	"net"
	"time"

	"k8s.io/apimachinery/pkg/util/rand"
)

type RegistryGroupController struct {
	beego.Controller
}

// 仓库组入口页面
// @router /image/registry/group/list [get]
func (this *RegistryGroupController) RegistryGroupList() {
	this.TplName = "image/registry/group/list.html"
}

// 2019-01-28 9:24
// 仓库组详情入口页面
// @route /image/registry/group/detail/:hi(.*) [get]
// @router /image/registry/group/detail/:id:int [get]
func (this *RegistryGroupController) GroupDetailPage() {
	registrData := getRegistryGroup(this)
	this.Data["ServiceName"] = registrData.ServerDomain

	reg := GetRegistryServerCluster(registrData.ServerDomain, registrData.ClusterName)
	if len(reg.ServerAddress) > 10 {
		s := strings.Split(reg.ServerAddress, ":")
		host := reg.ServerDomain + ":" + s[1]
		registrData.ServerDomain = host
	}

	this.Data["data"] = registrData
	this.TplName = "image/registry/group/detail.html"
}

// @router /image/registry/group/add [get]
func (this *RegistryGroupController) RegistryGroupAdd() {
	var clusterHtml string
	clusterData := cluster.GetClusterSelect()
	update := registry.CloudRegistryGroup{}
	update.GroupType = "公开"

	id := this.GetString("GroupId")
	this.Data["GroupType1"] = ""
	this.Data["GroupType2"] = "checked"

	// 更新操作
	if id != "0" {
		searchMap := sql.GetSearchMap("GroupId", *this.Ctx)
		q := sql.SearchSql(registry.CloudRegistryGroup{},
			registry.SelectCloudRegistryGroup,
			searchMap)
		sql.Raw(q).QueryRow(&update)

		clusterHtml = util.GetSelectOptionName(update.ClusterName)
		this.Data["registryHtml"] = util.GetSelectOptionName(update.ServerDomain)

		if update.GroupType == "私有" {
			this.Data["GroupType1"] = "checked"
			this.Data["GroupType2"] = ""
		}
	}

	this.Data["data"] = update
	this.Data["cluster"] = clusterHtml + clusterData
	this.TplName = "image/registry/group/add.html"
}

// json
// @router /api/registry/group [post]
func (this *RegistryGroupController) SaveRegistryGroup() {
	d := registry.CloudRegistryGroup{}
	err := this.ParseForm(&d)
	if err != nil {
		this.Ctx.WriteString("参数错误" + err.Error())
		return
	}

	searchMap := sql.SearchMap{}
	searchMap.Put("GroupId", d.GroupId)
	searchMap.Put("CreateUser", getUser(this))

	masterData := make([]registry.CloudRegistryGroup, 0)
	q := sql.SearchSql(d, registry.SelectCloudRegistryGroup, searchMap)
	sql.Raw(q).QueryRows(&masterData)
	util.SetPublicData(d, getUser(this), &d)

	q = sql.InsertSql(d, registry.InsertCloudRegistryGroup)
	if d.GroupId > 0 {
		q = sql.UpdateSql(d, registry.UpdateCloudRegistryGroup, searchMap, registry.UpdateGroupExclude)
	} else {
		status, msg := checkQuota(getUser(this))
		if !status {
			data := util.ApiResponse(false, msg)
			setJson(this, data)
			return
		}
	}

	_, err = sql.Raw(q).Exec()
	data, msg := util.SaveResponse(err, "名称已经被使用")
	util.SaveOperLog(getUser(this), *this.Ctx, "操作仓库组 "+msg, d.GroupName)
	setJson(this, data)
}

// 2019-02-12 08:40
// 检查镜像仓库配额
// 检查资源配额是否够用
func checkQuota(username string) (bool, string) {
	quotaDatas := quota.GetUserQuotaData(username, "")
	for _, v := range quotaDatas {
		if v.RegistryGroupUsed+1 > v.RegistryGroupNumber {
			return false, "仓库组数量超过配额限制"
		}
	}
	return true, ""
}

// Registry group image log
// @param group
// 2019-01-29 16:41
// @router /api/registry/group/images/log [get]
func (this *RegistryGroupController) RegistryImagesLog() {
	data := make([]registry.CloudImageLog, 0)

	searchMap := sql.GetSearchMapV("RepositoriesGroup",
		this.GetString("GroupName"),
		"ClusterName",
		this.GetString("ClusterName"),
	)

	key := this.GetString("search")
	searchSql := sql.SearchSql(registry.CloudImageLog{}, registry.SelectCloudImageLog, searchMap)
	if key != "" {
		key = sql.Replace(key)
		searchSql += strings.Replace(registry.SelectImageLogWhere, "?", key, -1)
	}

	num, err := sql.OrderByPagingSql(searchSql,
		"create_time",
		*this.Ctx.Request,
		&data,
		registry.CloudImageLog{})

	r := util.ResponseMap(data,
		sql.Count("cloud_image_log", int(num), key),
		this.GetString("draw"))
	if err != nil {
		r = util.ResponseMapError(err.Error())
	}
	setJson(this, r)
}

// 仓库组镜像数据
// @router /api/registry/group/images [get]
func (this *RegistryGroupController) RegistryGroupImages() {
	data := make([]k8s.CloudImage, 0)
	searchMap := sql.SearchMap{}
	group := this.GetString("GroupName")
	searchMap.Put("RepositoriesGroup", group)
	key := this.GetString("search")
	searchSql := sql.SearchSql(k8s.CloudImage{}, registry.SelectCloudImage, searchMap)

	if key != "" {
		key = sql.Replace(key)
		searchSql += strings.Replace(registry.SelectCloudImageWhere, "?", key, -1)
	}

	harborClient := harbor.NewClient(nil, "https://reg.testcloud.com", "admin", "Harbor12345")
	// Project Name in Harbor
	opt := &harbor.ListProjectsOptions{Name: group}
	projects, _, errs := harborClient.Projects.ListProject(opt)
	if errs == nil {
		logs.Info(projects)
	}
	repoOpt := &harbor.ListRepositoriesOption{ProjectId: projects[0].ProjectID}

	repos, _, reperr := harborClient.Repositories.ListRepository(repoOpt)

	if reperr == nil {
		logs.Info(repos)
	}

	num, err := sql.OrderByPagingSql(searchSql,
		"create_time",
		*this.Ctx.Request,
		&data,
		k8s.CloudImage{})

	if num < len(repos) {
		//d := k8s.CloudImage{}
		// imageSearchMap := sql.SearchMap{}
		// imageSearchMap.Put("GroupId", d.GroupId)
		// imageSearchMap.Put("CreateUser", getUser(this))

		// masterData := make([]registry.CloudRegistryGroup, 0)
		// q := sql.SearchSql(d, registry.SelectCloudRegistryGroup, searchMap)
		// sql.Raw(q).QueryRows(&masterData)
		// util.SetPublicData(d, getUser(this), &d)

		// q = sql.InsertSql(d, registry.InsertCloudImage)
		// if d.GroupId > 0 {
		// 	q = sql.UpdateSql(d, registry.UpdateCloudImage, searchMap, registry.UpdateCloudImageExclude)
		// }
		// _, err = sql.Raw(q).Exec()
	}
	r := util.ResponseMap(data,
		sql.Count("cloud_image", int(num), key),
		this.GetString("draw"))
	if err != nil {
		r = util.ResponseMapError(err.Error())
	}
	setJson(this, r)
}

// 获取认证服务器IP地址
func getAuthServer(authServer string) (string, string) {
	authServer = strings.Split(authServer, "/")[2]
	authServer = strings.Split(authServer, ":")[0]
	ns, _ := net.LookupHost(authServer)
	if len(ns) > 0 {
		return ns[0], authServer
	}
	return authServer, authServer
}

// 获取组数据
// 2019-01-31 21:03
func GetRegistryGroup(groupName string, clusterName string) (registry.CloudRegistryServer, string, string, string) {
	data := registry.CloudRegistryServer{}
	q := registry.SelectRegistryServerGroup
	q = strings.Replace(q, "{0}", sql.Replace(groupName), -1)
	q = strings.Replace(q, "{1}", sql.Replace(clusterName), -1)
	searchSql := sql.SearchSql(registry.CloudRegistryServer{}, q, sql.SearchMap{})
	sql.Raw(searchSql).QueryRow(&data)
	client, _ := k8s.GetClient(clusterName)
	nodes := k8s.GetNodesIp(client)
	logs.Info("执行job获取到节点地址", util.ObjToString(nodes))
	ip, domain := getAuthServer(data.AuthServer)
	return data, nodes[rand.Intn(len(nodes)-1)].Ip, ip, domain
}

// 仓库组数据
// @router /api/registry/group [get]
func (this *RegistryGroupController) RegistryGroup() {
	data := make([]registry.CloudRegistryGroup, 0)
	searchMap := sql.SearchMap{}
	groupTp := this.GetString("groupType")
	clusterMame := this.GetString("ClusterName")

	if clusterMame != "" {
		searchMap.Put("ClusterName", clusterMame)
		//searchMap.Put("CreateUser", getUser(this))
	}

	key := this.GetString("search")
	if groupTp == "公开" {
		searchMap.Put("GroupType", "公开")
	}
	if groupTp == "我的" {
		searchMap.Put("CreateUser", getUser(this))
	}
	searchSql := sql.SearchSql(registry.CloudRegistryGroup{}, registry.SelectCloudRegistryGroup, searchMap)
	if key != "" {
		key = sql.Replace(key)
		searchSql += strings.Replace(registry.SelectCloudRegistryGroupWhere, "?", key, -1)
	}

	user := getUser(this)
	perm := userperm.GetResourceName("镜像仓库组", user)

	num, err := sql.OrderByPagingSql(searchSql,
		"create_time",
		*this.Ctx.Request,
		&data,
		registry.CloudRegistryGroup{})

	result := make([]registry.CloudRegistryGroup, 0)
	for _, v := range data {
		if v.CreateUser != user && v.GroupType != "公开" {
			if !userperm.CheckPerm(v.GroupName, v.ClusterName, "", perm) {
				continue
			}
		}
		result = append(result, v)
	}
	r := util.ResponseMap(result, num, this.GetString("draw"))
	if err != nil {
		r = util.ResponseMapError(err.Error())
	}
	go UpdateGroupImageInfo()
	setJson(this, r)
}

// 2019-02-07 08:26
// 获取selecthtml
func GetRegistryGroupSelect(user string) string {
	html := make([]string, 0)
	data := []registry.CloudRegistryGroup{}
	q := sql.SearchSql(registry.CloudRegistryGroup{},
		registry.SelectCloudRegistryGroup,
		sql.GetSearchMapV("CreateUser", user))
	sql.Raw(q).QueryRows(&data)
	for _, v := range data {
		html = append(html, util.GetSelectOptionName(v.GroupName))
	}
	return strings.Join(html, "\n")
}

// 2019-02-07 08:32
// 获取selecthtml
func GetImageSelect(searchMap sql.SearchMap) string {
	html := make([]string, 0)
	data := GetImageDatas(searchMap)
	for _, v := range data {
		html = append(html, util.GetSelectOptionName(v.Name))
	}
	return strings.Join(html, "\n")
}

// 2019-02-07 8;39
// 获取版本号select
func GetImageTagSelect(tag string) string {
	tags := strings.Split(tag, ",")
	html := make([]string, 0)
	for _, v := range tags {
		html = append(html, v)
	}
	return strings.Join(html, "\n")
}

// 2019-01-28 10:33
func getRegistryGroup(this *RegistryGroupController) registry.CloudRegistryGroup {
	searchMap := sql.GetSearchMap("GroupId", *this.Ctx)
	imageName := this.Ctx.Input.Param(":hi")
	if len(imageName) > 0 {
		searchMap.Put("GroupName", imageName)
	}
	registryData := registry.CloudRegistryGroup{}
	q := sql.SearchSql(registryData, registry.SelectCloudRegistryGroup, searchMap)
	sql.Raw(q).QueryRow(&registryData)
	return registryData
}

// @router /api/registry/group [delete]
func (this *RegistryGroupController) DeleteRegistryGroup() {
	searchMap := sql.GetSearchMap("GroupId", *this.Ctx)
	registryData := getRegistryGroup(this)
	q := sql.DeleteSql(registry.DeleteCloudRegistryGroup, searchMap)
	r, _ := sql.Raw(q).Exec()
	data := util.DeleteResponse(nil,
		*this.Ctx, "删除仓库组,名称:"+registryData.GroupName,
		getUser(this),
		registryData.GroupName, r)
	setJson(this, data)
}

// 记录删除镜像审计操作
// 2019-01-29 08:57
func deleteImageLog(img k8s.CloudImage, this *RegistryGroupController, clustername string) {
	imglog := registry.CloudImageLog{}
	imglog.CreateUser = getUser(this)
	imglog.Ip = util.GetClientIp(this.Ctx.Request)
	imglog.CreateTime = util.GetDate()
	imglog.ClusterName = clustername
	imglog.Repositories = img.Access
	imglog.Name = img.Name
	imglog.OperType = "delete"
	imglog.RepositoriesGroup = img.RepositoriesGroup
	q := sql.InsertSql(imglog, registry.InsertCloudImageLog)
	sql.Raw(q).Exec()
}

// 2019-01-29 10:57
// 获取镜像数据
// @router /api/registry/group/images/:id:int [delete]
func (this *RegistryGroupController) GetRegistryGroupImage() {
	data := getImageData(this)
	r := util.ResponseMap(data, 1, 1)
	setJson(this, r)
}

// 2019-02-07 8:30
// 获取镜像数据
func GetImageDatas(searchMap sql.SearchMap) []k8s.CloudImage {
	imgData := make([]k8s.CloudImage, 0)
	q := sql.SearchSql(k8s.CloudImage{}, registry.SelectCloudImage, searchMap)
	sql.Raw(q).QueryRows(&imgData)
	return imgData
}

// 获取镜像数据
func getImageData(this *RegistryGroupController) k8s.CloudImage {
	searchMap := sql.SearchMap{}
	imageName := this.Ctx.Input.Param(":hi")
	if len(imageName) > 0 {
		searchMap.Put("Name", imageName)
		searchMap.Put("RepositoriesGroup", this.GetString("GroupName"))
	} else {
		sql.GetSearchMap("ImageId", *this.Ctx)
	}
	logs.Info("searchMap", searchMap)
	imgData := GetImageDatas(searchMap)
	if len(imgData) > 0 {
		return imgData[0]
	}
	return k8s.CloudImage{}
}

// 2019-01-29 8:51
// 删除仓库组中的镜像
// @param tag
// @param force 删除数据库里的数据
// @router /api/registry/group/images/:id:int [delete]
func (this *RegistryGroupController) DeleteRegistryGroupImage() {
	force := this.GetString("force")
	searchMap := sql.GetSearchMap("ImageId", *this.Ctx)
	imgData := getImageData(this)
	server := GetRegistryServer(strings.Split(imgData.Access, ":")[0])
	if len(server) == 0 && force == "" {
		data := util.DeleteResponse(errors.UnsupportedError("没有知道对应的仓库服务"),
			*this.Ctx, "删除镜像,名称:"+imgData.Name,
			getUser(this),
			imgData.Name,
			driver.ResultNoRows)
		setJson(this, data)
		return
	}

	if len(server) > 0 {
		_, err := k8s.DeleteRegistryImage(imgData.Access,
			server[0].Admin,
			util.Base64Decoding(server[0].Password),
			imgData.Name,
			this.GetString("tag"))

		if err != nil && force == "" {
			data := util.DeleteResponse(err,
				*this.Ctx, "删除镜像,名称:"+imgData.Name,
				getUser(this),
				imgData.Name,
				driver.ResultNoRows)
			setJson(this, data)
			return
		}
	}

	q := sql.DeleteSql(registry.DeleteCloudImage, searchMap)
	dr, _ := sql.Raw(q).Exec()
	data := util.DeleteResponse(
		nil,
		*this.Ctx,
		"删除镜像,名称:"+imgData.Name,
		getUser(this),
		imgData.Name,
		dr)

	setJson(this, data)
	if len(server) > 0 {
		deleteImageLog(imgData, this, server[0].ClusterName)
	}
}

// 设置json数据
func setJson(this *RegistryGroupController, data interface{}) {
	this.Data["json"] = data
	this.ServeJSON(false)
}

// 2019-01-28 15:55
// 将已经存在的数据查到map,做更新或插入判断
func getExistsImageMap() util.Lock {
	existsImages := make([]k8s.CloudImage, 0)
	q := sql.SearchSql(k8s.CloudImage{}, registry.SelectCloudImageExists, sql.SearchMap{})
	sql.Raw(q).QueryRows(&existsImages)
	lock := util.Lock{}
	for _, v := range existsImages {
		lock.Put(v.RepositoriesGroup+v.Name, 1)
	}
	return lock
}

// 2019-02-09 14:05
// 获取镜像信息
func getRegistryInfo(data []registry.CloudRegistryGroup) util.Lock {
	lock := util.Lock{}
	for _, v := range data {
		key := v.ServerDomain + v.ClusterName
		if _, ok := lock.Get(key); !ok {
			reg := GetRegistryServerCluster(v.ServerDomain, v.ClusterName)
			if len(reg.ServerAddress) > 10 {
				s := strings.Split(reg.ServerAddress, ":")
				host := reg.ServerDomain + ":" + s[1]
				img, tag, imageInfo := k8s.GetRegistryInfo(host,
					reg.Admin,
					util.Base64Decoding(reg.Password),
					v.ServerDomain)
				lock.Put(key, []util.Lock{img, tag, imageInfo})
			}
		}
	}
	return lock
}

// 更新仓库组数据,镜像数量,tag数量
// 2019-01-27 21:07
var ImageDataUpdate util.Lock

// 2019-02-20 18:34
// 获取镜像下载日志信息
func getImageLogId(img k8s.CloudImage) registry.CloudImageLog {
	logData := registry.CloudImageLog{}
	key := img.Name + img.RepositoriesGroup
	r := cache.RegistryLogCache.Get(key)
	status := util.RedisObj2Obj(r, &logData)
	if !status {
		logq := sql.SearchSql(registry.CloudImageLog{},
			registry.SelectImageDownload,
			sql.GetSearchMapV("Name",
				img.Name,
				"RepositoriesGroup",
				img.RepositoriesGroup))
		sql.Raw(logq).QueryRow(&logData)
		cache.RegistryLogCache.Put(key, util.ObjToString(logData), time.Minute*10)
	}
	return logData
}

func UpdateGroupImageInfo() {
	if !util.WriteLock("lastUpdate", &ImageDataUpdate, 10) {
		logs.Info("更新仓库组信息间隔太小")
		return
	}
	data := make([]registry.CloudRegistryGroup, 0)
	searchMap := sql.SearchMap{}
	searchSql := sql.SearchSql(registry.CloudRegistryGroup{},
		registry.SelectCloudRegistryGroup,
		searchMap)

	sql.Raw(searchSql).QueryRows(&data)

	lock := getRegistryInfo(data)
	imageExists := getExistsImageMap()

	for _, v := range data {
		key := v.ServerDomain + v.ClusterName
		if _, ok := lock.Get(key); ok {
			d := lock.GetV(key).([]util.Lock)
			if len(d) > 2 {
				img := d[0]
				tag := d[1]
				imageInfo := d[2]
				for k, imgv := range img.GetData() {
					if k == v.GroupName {
						v.ImageNumber = int64(imgv.(int))
					}
				}
				for k, tagv := range tag.GetData() {
					if k == v.GroupName {
						v.TagNumber = int64(tagv.(int))
					}
				}
				for _, infov := range imageInfo.GetData() {
					img := infov.(k8s.CloudImage)
					q := sql.InsertSql(img, registry.InsertCloudImage)
					if _, ok := imageExists.Get(img.RepositoriesGroup + img.Name); ok {
						qmap := sql.GetSearchMapV("Name", img.Name, "RepositoriesGroup", img.RepositoriesGroup)
						logData := getImageLogId(img)
						img.Download = logData.LogId
						q = sql.UpdateSql(img,
							registry.UpdateCloudImage,
							qmap,
							registry.UpdateCloudImageExclude)
					}
					sql.Raw(q).Exec()
				}
				sqlMap := sql.SearchMap{}
				sqlMap.Put("GroupId", v.GroupId)
				u := sql.UpdateSql(
					v,
					registry.UpdateCloudRegistryGroup,
					sqlMap,
					registry.UpdateCloudRegistryGroupExclude)

				sql.Raw(u).Exec()
			}
		}
	}
}

// 2019-02-07 12:34
// 在部署时使用的镜像数据
// @router /api/registry/deploy/image [get]
func (this *RegistryGroupController) GetDeployImage() {
	data := make([]registry.CloudDeployImage, 0)
	user := getUser(this)
	//clusterName := this.GetString("clusterName")
	search := this.GetString("search[value]")

	q := sql.SearchSql(registry.CloudDeployImage{},
		registry.SelectDeployImage,
		sql.SearchMap{})

	q = strings.Replace(q, "{0}", sql.Replace(search), -1)
	sql.GetOrm().Raw(q, user).QueryRows(&data)

	result := make([]registry.CloudDeployImage, 0)
	for _, v := range data {
		temp := registry.CloudDeployImage(v)
		servers := strings.Split(v.ServerAddress, ":")
		if len(servers) > 0 {
			temp.ServerDomain = v.ServerDomain + ":" + servers[1]
		}
		result = append(result, temp)
	}
	r := util.ResponseMap(result, len(data), this.GetString("draw"))
	setJson(this, r)
}

// 2019-02-08 13:50
// 获取登录用户
func getUser(this *RegistryGroupController) string {
	return util.GetUser(this.GetSession("username"))
}

// 2019-02-14 19:02
// 获取发布服务时的镜像tag
func GetImageTag(images string) string {
	// 创建私有仓库镜像获取私密文件
	imgS := strings.Split(images, "/")
	if len(imgS) < 2 {
		return ""
	}
	names := strings.Join(imgS[1:], "/")
	name := strings.Split(names, ":")[0]
	searchMap := sql.GetSearchMapV("Name", name, "Access", imgS[0])
	r := GetImageDatas(searchMap)
	tags := strings.Split(r[0].Tags, ",")
	tagsTemp := make([]string, 0)
	for i := len(tags) - 1; i >= 0; i-- {
		tagsTemp = append(tagsTemp, util.GetSelectOptionName(tags[i]))
	}
	return strings.Join(tagsTemp, "")
}
