package main

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/astaxie/beego/logs"
	"github.com/gin-gonic/gin"
	"github.com/yulibaozi/beku"
	"gitlab.yc345.tv/backend/onion_util"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Body struct {
	Name                string            `json:"name,omitempty"`
	Port                int64             `json:"port,omitempty"`
	ConfigMapVolumePath string            `json:"configMapVolumePath,omitempty"`
	Image               string            `json:"image,omitempty"`
	ConfigValue         map[string]string `json:"configValue,omitempty"`
}

type App struct {
	namespace           string
	name                string
	configMapVolumePath string
	configValue         map[string]string
	ctime               string
	image               string
	port                int32
}

var clientset *kubernetes.Clientset

var once sync.Once

func ConfigInit() {
	once.Do(func() {
		LoadInconfig()
	})
}

func LoadInconfig() {
	ph, _ := os.Getwd()
	configPath := filepath.Join(ph, "config")
	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		panic(err)
	}
	var err1 error
	clientset, err1 = kubernetes.NewForConfig(config)
	if err1 != nil {
		panic(err1)
	}
	logs.Info("greate success")
}

func main() {
	ConfigInit()
	r := gin.Default()
	rt := r.Group(`/k8sdep`)
	rt.POST("/apply",
		onion_util.GRequestBodyObject(reflect.TypeOf(Body{}), "json"),
		Apply)
	r.Run(":5001")
}

func Apply(c *gin.Context) {
	body := c.MustGet("requestBody").(*Body)
	app := &App{
		namespace: "test",
		name:      body.Name,
		ctime:     time.Now().Format("2006-01-02 15:04:05"),
		image: func() string {
			if body.Image != "" {
				return body.Image
			}
			return fmt.Sprintf("registry.cn-hangzhou.aliyuncs.com/guanghe/%s:test", body.Name)
		}(),
		configMapVolumePath: func() string {
			if body.ConfigMapVolumePath != "" {
				return body.ConfigMapVolumePath
			}
			return "/etc/config"
		}(),
		configValue: func() map[string]string {
			if body.ConfigValue != nil {
				return body.ConfigValue
			}
			return nil
		}(),
		port: int32(body.Port),
	}
	if err := CreateOrUpdate(app); err != nil {
		c.JSON(200, gin.H{"msg": err.Error()})
		return
	}
	fmt.Println(app, app)
	c.JSON(200, gin.H{"msg": "OK"})
}

func CreateOrUpdate(app *App) error {
	logs.Info(app)
	logs.Info(1)
	_, err := clientset.CoreV1().Services(app.namespace).Get(app.name, metav1.GetOptions{})
	if err != nil {
		logs.Info("不存在 service: %s", app.name)
		if err := CreateSvc(app); err != nil {
			logs.Error(err.Error())
			return err
		}
	}
	logs.Info(2)
	if err := CreateOrUpdateConfigMap(app.namespace, app.name, app.configValue); err != nil {
		logs.Error(err.Error())
		return err
	}
	logs.Info(3)
	if err := CreateOrUpdateDep(app); err != nil {
		logs.Error(err.Error())
		return err
	}
	logs.Info(4)
	if err := PacthIngress(app); err != nil {
		logs.Error(err.Error())
		return err
	}
	logs.Info(5)
	return nil
}

func GetPodsByNs(ns string) ([]v1.Pod, error) {

	podList, e := clientset.CoreV1().Pods(ns).List(metav1.ListOptions{
		TypeMeta: metav1.TypeMeta{
			Kind:       "",
			APIVersion: "",
		},
		LabelSelector:   "",
		FieldSelector:   "",
		Watch:           false,
		ResourceVersion: "",
		TimeoutSeconds:  nil,
		Limit:           0,
		Continue:        "",
	})
	if e != nil {
		return nil, e
	}

	return podList.Items, nil
}

func PatchServiceAccount(nsname string) error {

	res, _ := clientset.CoreV1().ServiceAccounts(nsname).Get("default", metav1.GetOptions{})

	res.ImagePullSecrets = []v1.LocalObjectReference{{Name: "registry.cn-hangzhou.aliyuncs.com"}}

	r1, err := clientset.CoreV1().ServiceAccounts(nsname).Update(res)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Printf("%+v", r1)
	return nil
}

func Render(path string, config map[string]interface{}) ([]byte, error) {
	t, err := template.ParseFiles(path)
	if err != nil {
		logs.Error(err.Error())
		return nil, err
	}
	bufer := bytes.NewBuffer(nil)
	err = t.Execute(bufer, config)
	if err != nil {
		logs.Error(err.Error())
		return nil, err
	}
	return bufer.Bytes(), nil
}

func PacthIngress(app *App) error {
	ingress, err := clientset.ExtensionsV1beta1().Ingresses(app.namespace).Get("ladder", metav1.GetOptions{})
	if err != nil {
		logs.Error(err.Error())
		return err
	}

	pa := v1beta1.HTTPIngressPath{
		Path: fmt.Sprintf("/%s", app.name),
		Backend: v1beta1.IngressBackend{
			ServiceName: app.name,
			ServicePort: intstr.IntOrString{
				Type:   0,
				IntVal: app.port,
				StrVal: "",
			},
		},
	}
	exist := false
	for _, v := range ingress.Spec.Rules[0].IngressRuleValue.HTTP.Paths {
		if v.Backend.ServiceName == app.name {
			exist = true
			break
		}
	}

	if !exist {
		ingress.Spec.Rules[0].IngressRuleValue.HTTP.Paths = append(ingress.Spec.Rules[0].IngressRuleValue.HTTP.Paths, pa)
		update, err := clientset.ExtensionsV1beta1().Ingresses(app.namespace).Update(ingress)
		if err != nil {
			logs.Error(err.Error())
			return err
		}
		fmt.Println(update)
	}
	return nil
}

func CreateNs(name string) error {
	ns, _ := beku.NewNs().SetName(name).Finish()
	a, err := clientset.CoreV1().Namespaces().Create(ns)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println(a)
	return nil
}

func GetNs() ([]string, error) {
	res := []string{}
	list, e := clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
	if e != nil {
		return res, e
	}
	for _, v := range list.Items {
		res = append(res, v.GetName())
	}
	return res, nil
}

func DeleteNs(name string) error {
	err := clientset.CoreV1().Namespaces().Delete(name, nil)
	if err != nil {
		logs.Error(err.Error())
		return err
	}
	return nil
}

func CreateOrUpdateDep(app *App) error {
	imageSecret := strings.Split(app.image, "/")[0]
	sec, err := beku.NewDeployment().SetName(app.name).SetNamespace(app.namespace).
		SetLabels(map[string]string{"app": app.name}).SetSelector(map[string]string{"app": app.name}).
		SetContainer(app.name, app.image, app.port).SetPVCMounts(app.name, app.configMapVolumePath).
		SetImagePullSecrets(imageSecret).SetEnvs(map[string]string{"time": app.ctime}).Finish()
	sec.Spec.Template.Spec.Volumes = []v1.Volume{{
		Name: app.name,
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{Name: app.name},
			},
		},
	}}
	sec.Spec.Template.ObjectMeta.Labels = map[string]string{"app": app.name}
	containers := []v1.Container{}
	for _, v := range sec.Spec.Template.Spec.Containers {
		v.ImagePullPolicy = v1.PullAlways
		containers = append(containers, v)
	}
	sec.Spec.Template.Spec.Containers = containers
	yamlByts, err := beku.ToYAML(sec)
	if err != nil {
		panic(err)
	}
	fmt.Println("\n" + string(yamlByts))

	_, err = clientset.AppsV1().Deployments(app.namespace).Get(app.name, metav1.GetOptions{})
	if err != nil {
		logs.Info("不存在deploy")
		if _, err := clientset.AppsV1().Deployments(app.namespace).Create(sec); err != nil {
			logs.Error(err.Error())
			return err
		}
	} else {
		logs.Info("已经存在deploy")
		_, err := clientset.AppsV1().Deployments(app.namespace).Update(sec)
		if err != nil {
			logs.Error(err.Error())
			return err
		}
	}
	return nil
}

func CreateSvc(app *App) error {
	svc, err := beku.NewSvc().SetNamespaceAndName(app.namespace, app.name).SetLabels(map[string]string{"app": app.name}).
		SetSelector(map[string]string{"app": app.name}).SetServiceType(beku.ServiceTypeClusterIP).
		SetPort(beku.ServicePort{Port: app.port}).Finish()
	if err != nil {
		logs.Error(err.Error())
		return err
	}

	_, err1 := clientset.CoreV1().Services(app.namespace).Create(svc)
	if err1 != nil {
		logs.Error(err1.Error())
		return err1
	}
	return nil
}

func CreateSecret(nsname, path string) error {
	b, err := Render(path, map[string]interface{}{"namespace": nsname})
	if err != nil {
		logs.Error("path 指定错误")
		return err
	}

	sec, err := beku.NewSecret().YAMLNew(b).Finish()
	if err != nil {
		logs.Error(err.Error())
		return err
	}
	fmt.Println(sec)
	yamlByts, err := beku.ToYAML(sec)
	if err != nil {
		logs.Error("paser yaml 错误")
		return err
	}
	fmt.Println("\n" + string(yamlByts))

	clientset.CoreV1().Secrets(nsname).Create(&v1.Secret{})
	a, err := clientset.CoreV1().Secrets(nsname).Create(sec)
	if err != nil {
		logs.Error(err.Error())
		return err
	}
	fmt.Println(a)
	return nil

}

func CreateOrUpdateConfigMap(nsname, name string, config map[string]string) error {
	isNil := false
	if config == nil {
		config = map[string]string{"env": "development"}
		isNil = true
	}
	sec, err := beku.NewCM().SetName(name).SetNamespace(nsname).SetData(config).Finish()
	if err != nil {
		logs.Error(err.Error())
		return err
	}
	yamlByts, err := beku.ToYAML(sec)
	if err != nil {
		logs.Error("paser yaml 错误")
		return err
	}
	fmt.Println("configMap: \n" + string(yamlByts))

	_, err = clientset.CoreV1().ConfigMaps(nsname).Get(name, metav1.GetOptions{})
	if err != nil {
		_, err := clientset.CoreV1().ConfigMaps(nsname).Create(sec)
		if err != nil {
			fmt.Println(err)
			return err
		}
	} else {
		if isNil {
			return nil
		}
		_, err := clientset.CoreV1().ConfigMaps(nsname).Update(sec)
		if err != nil {
			fmt.Println(err)
			return err
		}
	}

	return nil
}