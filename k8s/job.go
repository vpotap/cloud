package k8s

import (
	"github.com/astaxie/beego/logs"
	"encoding/json"
	"strings"
	"cloud/util"
	"k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"time"
	"cloud/cache"
	"github.com/astaxie/beego"
)

// docker build -t ITEMNAME:VERSION -f  /root/docker /root --ulimit nproc=2048:4096  --ulimit nofile=4096:10000
var buildCmd = `
d=$(date +"%F %T")
echo "开始构建...$d"
echo "构建服务器$HOSTNAME"
echo REGISTRYIP REGISTRYDOMAIN >> /etc/hosts
echo AuthServerIp AuthServerDomain >> /etc/hosts
ping REGISTRYDOMAIN -c 1
ping AuthServerDomain -c 1
dockerd --ip-forward=false --iptables=false --insecure-registry REGISTRY  --storage-driver=devicemapper  &>/dev/null &
d=$(date +"%F %T")
echo "$d 开始编译文件,,,,"
SCRIPT
d=$(date +"%F %T")
echo "$d 编译完成..."
set +x
sleep 10
echo $DOCKERFILE
seq="1 2 3 4 5 6 7 8 9 10"

for i in $seq
do
  docker ps |grep IMAGE && break
  sleep 3
  d=$(date +"%F %T")
  echo "请等待.. $i  $d" 
done

cat > /root/docker <<EOF
$DOCKERFILE
EOF
cat /root/docker
echo ""
d=$(date +"%F %T")
echo "开始构建... $d"
for i in $seq
do
   docker build -t REGISTRY/REGISTRYGROUP/ITEMNAME:VERSION -f  /root/docker /root --ulimit nproc=MINPROC:MAXPROC  --no-cache --ulimit nofile=MINFILE:MAXFILE  2>&1
   if [ $? -eq 0 ] ; then
      break
   fi
done
if [ $? -eq 0 ] ; then
echo "镜像信息"
docker images |grep REGISTRY/REGISTRYGROUP/ITEMNAME |grep VERSION
if [ $? -gt 0 ] ; then
         d=$(date +"%F %T")
         echo "完成构建...$d"
         echo "构建失败"
         echo "构建镜像失败了..$d" 
         echo "完成构建...$d"
         exit 0
fi
mkdir /root/.docker -p
cat > /root/.docker/config.json <<EOF
{
        "auths": {
                "REGISTRY": {
                        "auth": "AUTH"
                }
        },
        "HttpHeaders": {
                "User-Agent": "Docker-Client/18.01.0-ce (linux)"
        }
}
EOF
chmod 700 /root/.docker -R
   docker push REGISTRY/REGISTRYGROUP/ITEMNAME:VERSION 2>&1
   if [ $? -eq 0 ] ; then
          sync 2>/dev/null
          d=$(date +"%F %T")
          echo "完成构建...$d"
          echo "构建完成"
          echo "构建成功"
   else
   sleep 5000
         d=$(date +"%F %T")
         echo "完成构建...$d"
         echo "构建失败"
         echo "完成构建...$d"
   fi
else
   echo "构建失败..."
   d=$(date +"%F %T")
   echo "完成构建... $d"
   sleep 5
fi
exit 0
`

// 2019-01-25 10:51
type JobParam struct {
	// job 名称
	Jobname string
	// 执行命令
	Command []string
	// 超时时间
	Timeout int
	// master地址
	Master string
	// master 端口
	Port string
	// docker file
	Dockerfile string
	// 镜像仓库 // 编译完提交镜像
	RegistryServer string
	// 限制进程数据
	NoProcMax string
	NoProcMin string
	// 限制文件数据
	NoFileMax string
	NoFileMin string
	//  项目名称
	Itemname string
	// 版本
	Version string
	// 仓库认证密码
	Auth string
	// namespace
	Namespace string
	// 镜像服务域名
	RegistryDomain string
	// 镜像服务IP地址
	RegistryIp string
	// 仓库组地址
	RegistryGroup string
	// 镜像地址
	Images string
	// 配置文件
	ConfigureData []ConfigureData
	// job分配cpu
	Cpu int
	// 内存分配大小
	Memory int
	// 不能创建或更新configmap
	NoUpdateConfigMap bool
	// 私有仓库地址
	RegistryAuth string
	// 集群名称
	ClusterName string
	// 认证服务器IP地址
	AuthServerIp string
	// 认证服务器域名
	AuthServerDomain string
	// 构建脚本
	Script string
	// 环境变量
	Env string
	// 类型
	Type string
	// 服务器地址
	ServerAddress string
}

// 替换buildcmd
// 2019-01-26 12:38
func replace(s string, old string, new string) string {
	return strings.Replace(s, old, new, -1)
}

// 设置默认参数
// 2019-01-25 13:02
func setJobInitParam(param JobParam) JobParam {
	if param.NoProcMax == "" {
		param.NoProcMax = "4096"
		param.NoProcMin = "2048"
	}
	if param.NoFileMin == "" {
		param.NoFileMin = "4096"
		param.NoFileMax = "10000"
	}
	if param.Timeout == 0 {
		param.Timeout = 60
	}
	if param.Namespace == "" {
		param.Namespace = util.Namespace("job", "job")
	}
	if param.Images == "" {
		param.Images = "docker"
	}
	if len(param.Command) == 0 {
		param.Command = []string{"sh", "/build/build-cmd", ";", "exit", "0"}
	}
	if param.Cpu == 0 {
		param.Cpu = 2
	}
	if param.Memory == 0 {
		param.Memory = 4096
	}
	return param
}

// 获取build参数
// 2019-01-25 16:32
func getBuild(param JobParam) string {
	build := replace(buildCmd, "MAXFILE", param.NoFileMax)
	build = replace(build, "MINFILE", param.NoFileMin)
	build = replace(build, "SCRIPT", replaceVar(param.Itemname, param.Script))
	build = replace(build, "MINPROC", param.NoProcMin)
	build = replace(build, "MAXPROC", param.NoProcMax)
	build = replace(build, "ITEMNAME", param.Itemname)
	build = replace(build, "VERSION", param.Version)
	build = replace(build, "REGISTRYIP", param.RegistryIp)
	build = replace(build, "REGISTRYDOMAIN", param.RegistryDomain)
	build = replace(build, "REGISTRYGROUP", param.RegistryGroup)
	build = replace(build, "REGISTRY", param.RegistryServer)
	build = replace(build, "AuthServerIp", param.AuthServerIp)
	build = replace(build, "AuthServerDomain", param.AuthServerDomain)
	build = replace(build, "AUTH", param.Auth) // base64 user:passwd
	return build
}

// 获取编译配置文件
// 2019-01-25 16:34
func getBuildConfigdata(param JobParam) []ConfigureData {
	// 配置信息
	config := `[{"ContainerPath":"/build/","DataName":"build-job-` + param.Itemname + `","DataId":"build-cmd"}]`
	// 生产configmap信息
	configData := make([]ConfigureData, 0)
	json.Unmarshal([]byte(config), &configData)

	configureData := make([]ConfigureData, 0)
	for _, v := range configData {
		ConfigDbData := map[string]interface{}{
			"build-cmd": getBuild(param), // 启动命令
		}
		v.ConfigDbData = ConfigDbData
		configureData = append(configureData, v)
	}
	return configureData
}

// 转换参数
// 2019-01-25 16:37
func getJobParam(conf map[string]interface{}) v1.Job {
	job := v1.Job{}
	t1, _ := json.Marshal(conf)
	json.Unmarshal(t1, &job)
	return job
}

// 物理机系统和job系统必须一致
// 获取配置server创建所需参数
// 2019-01-25 16:42
func getJobServerParam(param JobParam) ServiceParam {
	serviceParam := ServiceParam{}
	dir := beego.AppConfig.String("docker.data.dir") + `data/source/`
	installDir := beego.AppConfig.String("docker.install.dir")
	serviceParam.StorageData = `[
        {"ContainerPath":"` + installDir + `","Volume":"","HostPath":"` + installDir + `"},
        {"ContainerPath":"` + dir + `","Volume":"","HostPath":"` + dir + `"},
        {"ContainerPath":"/var/run/docker.sock","Volume":"","HostPath":"/var/run/docker.sock"}, 
        {"ContainerPath":"/usr/bin/docker","Volume":"","HostPath":"/usr/bin/docker"},
        {"ContainerPath":"/etc/resolv.conf","Volume":"","HostPath":"/etc/resolv.conf"}]`
	serviceParam.Namespace = param.Namespace
	if len(param.ConfigureData) == 0 {
		configureData := getBuildConfigdata(param)
		serviceParam.ConfigureData = configureData
	}
	if len(param.Env) > 0 {
		serviceParam.Envs = param.Env
	}
	return serviceParam
}

// 获取是否要再指定label的机器构建
// 2019-01-25 16;51
func getJobLables(conf map[string]interface{}, clientSet kubernetes.Clientset) map[string]interface{} {
	// 获取是否标签有ci的,有的话就去有标签的构建
	nodes := GetNodes(clientSet, "ci=build")
	if len(nodes) > 0 {
		selector := `{"Lables":"ci","Value":"build"}`
		nodeSelector := getNodeSelectorNode(selector)
		conf["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["nodeSelector"] = nodeSelector
	}
	return conf
}

// 替换环境变量
// 替换项目名称
func replaceVar(item string, str string)  string{
  str = strings.Replace(str, "$item", item, -1)
  return str
}

// 创建job，主要在构建时使用
// 2019-01-25 10:41
func CreateJob(param JobParam) string {

	param = setJobInitParam(param)
	serviceParam := getJobServerParam(param)

	if len(param.ConfigureData) > 0 {
		serviceParam.ConfigureData = param.ConfigureData
	}

	selector := map[string]interface{}{
		"kubernetes.io/hostname": param.ServerAddress,
	}

	envList := getEnv(serviceParam.Envs)

	env := map[string]interface{}{
		"name":  "DOCKERFILE",
		"value": replaceVar(param.Itemname, param.Dockerfile),
	}
	envList = append(envList, env)
	logs.Info("job获取到环境变量", util.ObjToString(envList))

	volumes, volumeMounts := getVolumes(serviceParam.StorageData, serviceParam.ConfigureData, serviceParam)
	conf := map[string]interface{}{
		"apiVersion": "batch/v1",
		"kind":       "Job",
		"metadata": map[string]interface{}{
			"name": param.Jobname,
		},
		"spec": map[string]interface{}{
			"backoffLimit": 1,
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": param.Jobname,
				},
				"spec": map[string]interface{}{

					"containers": []map[string]interface{}{
						map[string]interface{}{
							"name":            param.Jobname,
							"image":           param.Images,
							"imagePullPolicy": "IfNotPresent",
							"command":         param.Command,
							"volumeMounts":    volumeMounts,
							"securityContext": map[string]interface{}{
								"capabilities": map[string]interface{}{},
								"privileged":   true,
							},

							"resources": map[string]interface{}{
								"limits": map[string]interface{}{
									"memory": strconv.Itoa(param.Memory) + "Mi",
									"cpu":    param.Cpu,
								},
								"requests": map[string]interface{}{
									"memory": strconv.Itoa(param.Memory) + "Mi",
									"cpu":    param.Cpu,
								},
							},
							"env": envList,
						},
					},
					"restartPolicy":         "OnFailure",
					"activeDeadlineSeconds": param.Timeout,
					"volumes":               volumes,
				},
			},
		},
	}

	if param.RegistryAuth != "" {
		serviceParam.RegistryAuth = "1"
		serviceParam.Registry = param.RegistryAuth
		conf = setImagePullPolice(serviceParam, conf)
	}

	logs.Info("获取执行job集群地址", util.ObjToString(param))
	clientSet, err := GetClient(param.ClusterName)
	if err != nil {
		logs.Error("获取客户端失败", err.Error())
	}

	CreateServiceAccount(clientSet, param.Namespace, "default")
	cl2, _ := GetYamlClient(param.ClusterName, "", "v1", "api")
	serviceParam.Cl2 = cl2
	serviceParam.Cl3 = clientSet

	secretsName := GetDockerImagePullName(strings.Split(param.Images, "/")[0])
	isExists := SecretIsExists(clientSet, param.Namespace, secretsName)
	if isExists{
		conf["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["imagePullSecrets"] = []map[string]interface{}{
			map[string]interface{}{
				"name": secretsName,
			},
		}
	}

	if ! param.NoUpdateConfigMap {
		logs.Info("获取到job的configmap", serviceParam.ConfigureData)
		CreateConfigmap(serviceParam)
	}

	conf = getJobLables(conf, clientSet)

	if len(param.ServerAddress) > 0 {
		conf["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["nodeSelector"] = selector
		logs.Info("添加选择器", util.ObjToString(selector))
	}

	obj := getJobParam(conf)
	deployments, err := clientSet.BatchV1().Jobs(param.Namespace).Create(&obj)

	if err != nil {
		logs.Error("创建失败 job ", err, deployments)
	}
	logs.Info("创建job", deployments)
	return param.Jobname
}

// 获取job的pod数据
// 2019-01-26 18:01
func getJobPod(pod string, cl kubernetes.Clientset, namespace string) ([]corev1.Pod, error) {
	listOpt := meta_v1.ListOptions{}
	listOpt.LabelSelector = "job-name=" + pod
	pods, err := cl.CoreV1().Pods(namespace).List(listOpt)
	if len(pods.Items) == 0 {
		pods, err := cl.CoreV1().Pods(namespace).Get(pod, meta_v1.GetOptions{})
		s := make([]corev1.Pod, 0)
		s = append(s, *pods)
		return s, err
	}
	return pods.Items, err
}

// 获取构建日志
// 2019-01-25 16:55
//cl,_ := k8s.GetClient("10.16.55.114","8080")
//k8s.GetJobLogs(cl, "job-33e87d842bb7712c9688ed4f99c94336-ss8hw")

func GetJobLogs(cl kubernetes.Clientset, pod string, namespace string, line int64 ) string {
	podsData := make([]corev1.Pod, 0)
	var err error
	if line  > 200000 {
		// 大概4M左右
		line = 200000
	}
	if cache.PodCache != nil {
		r := cache.PodCache.Get(pod + namespace)
		s := util.RedisObj2Obj(r, &podsData)
		if s {
			logs.Info("从redis获取到构建任务pod", len(podsData), " ", pod+namespace)
		}
	}
	if len(podsData) == 0 {
		logs.Info(pod, namespace)
		podsData, err = getJobPod(pod, cl, namespace)
		if err != nil{
			return err.Error()
		}
		if cache.PodCache != nil && len(podsData) > 0 {
			cache.PodCache.Put(pod+namespace, util.ObjToString(podsData), time.Second*500)
		}
	}
	logs.Info("获取到pod", podsData[0].Name)
	if len(podsData) > 0 && err == nil {
		opt := corev1.PodLogOptions{}
		//bt := int64(1024 * 1024 * 2)
		//opt.LimitBytes = &bt
		line := line
		opt.TailLines = &line
		r := cl.CoreV1().Pods(namespace).GetLogs(podsData[0].Name, &opt)
		c := r.Do()
		l, _ := c.Raw()
		if c.Error() == nil {
			return string(l)
		}
	}
	return ""
}

// 构建完成后删除job
// 2019-01-26 16:34
func DeleteJob(clientSet kubernetes.Clientset, jobName string, namespace string) {
	if namespace == "" {
		namespace = util.Namespace("job", "job")
	}
	err := clientSet.BatchV1().Jobs(namespace).Delete(jobName, &meta_v1.DeleteOptions{})
	if err != nil {
		logs.Error("删除构建job失败", err)
	}

	podData, err := getJobPod(jobName, clientSet, namespace)
	if len(podData) > 0 && err == nil {
		err = clientSet.CoreV1().Pods(namespace).Delete(podData[0].Name, &meta_v1.DeleteOptions{})
		if err != nil {
			logs.Error("删除构建pod失败", err)
		}
	}
}

// 2019-01-03 6:31
// 获取job执行计划结果删除job
func getJobResult(jobParam JobParam, keyword string, timeout int, logtp string) string {
	cl, _ := GetClient(jobParam.ClusterName)
	count := 0
	for {
		log := GetJobLogs(cl, jobParam.Jobname, jobParam.Namespace, 200000)
		logs.Info("获取到log", log)
		if strings.Contains(log, keyword) || count > timeout {
			DeleteJob(cl, jobParam.Jobname, jobParam.Namespace)
			if logtp == "nginx" {
				return getNginxJobLog(log)
			} else {
				return log
			}
		}
		count += 1
		time.Sleep(time.Second * 1)
	}
}

// 清除无效的任务计划
// 构建完成后删除job
// 2019-01-26 16:34
func ClearJob(clientSet kubernetes.Clientset) {
	namespace := util.Namespace("job", "job")
	jobs, err := clientSet.BatchV1().Jobs(namespace).List(meta_v1.ListOptions{})
	if err != nil {
		logs.Error("删除job获取列表失败", err)
	}
	if len(jobs.Items) == 0 {
		return
	}

	// 删除pod
	for _, v := range jobs.Items {
		pod, err := getJobPod(v.Name, clientSet, namespace)
		if len(pod) > 0 && err == nil {
			if pod[0].Status.Phase == "Failed" || pod[0].Status.Phase == "Succeeded" || pod[0].Status.Phase == "Unknown" {
				err = clientSet.CoreV1().Pods(namespace).Delete(pod[0].Name, &meta_v1.DeleteOptions{})

			}
			if pod[0].Status.Phase == "Pending" {
				if time.Now().Unix()-pod[0].CreationTimestamp.Unix() > 3000 {
					err = clientSet.CoreV1().Pods(namespace).Delete(pod[0].Name, &meta_v1.DeleteOptions{})
				}
			}
			if err != nil {
				logs.Error("删除构建pod失败", err)
			} else {
				logs.Info("删除job中的pod成功", util.ObjToString(pod))
			}
		}
	}

	// 删除job
	for _, v := range jobs.Items {
		if v.Status.Failed == 1 || v.Status.Succeeded == 1 ||  v.Status.Failed == 2 {
			err := clientSet.BatchV1().Jobs(namespace).Delete(v.Name, &meta_v1.DeleteOptions{})
			if err != nil {
				logs.Error("删除构建job失败", err)
			} else {
				logs.Info("删除job成功", util.ObjToString(v))
			}
		}
	}

}
