module github.com/openshift/appliance

go 1.22.1

toolchain go1.22.4

require (
	github.com/alecthomas/units v0.0.0-20240927000941-0f3dac36c52b
	github.com/blang/semver v3.5.1+incompatible
	github.com/briandowns/spinner v1.23.1
	github.com/cavaliergopher/grab/v3 v3.0.1
	github.com/containers/image v3.0.2+incompatible
	github.com/coreos/ignition/v2 v2.20.0
	github.com/coreos/stream-metadata-go v0.4.5
	github.com/diskfs/go-diskfs v1.4.0
	github.com/dustin/go-humanize v1.0.1
	github.com/go-openapi/swag v0.23.0
	github.com/golang/mock v1.7.0-rc.1
	github.com/hashicorp/go-version v1.7.0
	github.com/itchyny/gojq v0.12.16
	github.com/onsi/ginkgo/v2 v2.19.1
	github.com/onsi/gomega v1.34.0
	github.com/openconfig/goyang v1.6.0
	github.com/openshift/api v0.0.0-20240808203820-e69593239e49
	github.com/openshift/assisted-image-service v0.0.0-20241104223410-cc6cb6edc250
	github.com/openshift/assisted-service/api v0.0.0
	github.com/openshift/hive/apis v0.0.0-20231220215202-ad99b9e52d27
	github.com/openshift/installer v1.4.17
	github.com/openshift/machine-config-operator v0.0.1-0.20201023110058-6c8bd9b2915c
	github.com/pelletier/go-toml v1.9.5
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.9.4-0.20230606125235-dd1b4c2e81af
	github.com/spf13/cobra v1.8.1
	github.com/thedevsaddam/retry v1.2.1
	github.com/thoas/go-funk v0.9.3
	golang.org/x/crypto v0.29.0
	golang.org/x/term v0.26.0
	k8s.io/api v0.30.3
	k8s.io/apimachinery v0.30.3
	sigs.k8s.io/yaml v1.4.0
)

require (
	cloud.google.com/go/auth v0.7.2 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.3 // indirect
	cloud.google.com/go/compute/metadata v0.5.0 // indirect
	github.com/AlecAivazis/survey/v2 v2.3.5 // indirect
	github.com/Azure/azure-sdk-for-go v68.0.0+incompatible // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.12.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.6.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.9.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5 v5.7.0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20230124172434-306776ec8161 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.29 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.23 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/autorest/to v0.4.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.2.2 // indirect
	github.com/BurntSushi/toml v1.4.0 // indirect
	github.com/IBM-Cloud/bluemix-go v0.0.0-20211102075456-ffc4e11dfb16 // indirect
	github.com/IBM-Cloud/power-go-client v1.8.3 // indirect
	github.com/IBM/go-sdk-core/v5 v5.18.1 // indirect
	github.com/IBM/keyprotect-go-client v0.12.2 // indirect
	github.com/IBM/networking-go-sdk v0.45.0 // indirect
	github.com/IBM/platform-services-go-sdk v0.71.0 // indirect
	github.com/IBM/vpc-go-sdk v0.61.0 // indirect
	github.com/MakeNowJust/heredoc v1.0.0 // indirect
	github.com/PaesslerAG/gval v1.0.0 // indirect
	github.com/PaesslerAG/jsonpath v0.1.1 // indirect
	github.com/apparentlymart/go-cidr v1.1.0 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/aws/aws-sdk-go v1.55.5 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cavaliercoder/go-cpio v0.0.0-20180626203310-925f9528c45e // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/chai2010/gettext-go v1.0.2 // indirect
	github.com/cjlapao/common-go v0.0.39 // indirect
	github.com/clarketm/json v1.14.1 // indirect
	github.com/containers/image/v5 v5.31.0 // indirect
	github.com/containers/storage v1.54.0 // indirect
	github.com/coreos/go-json v0.0.0-20230131223807-18775e0fb4fb // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/coreos/vcontext v0.0.0-20230201181013-d72178a18687 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/digitalocean/go-libvirt v0.0.0-20240220204746-fcabe97a6eed // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/elliotwutingfeng/asciiset v0.0.0-20230602022725-51bbb787efab // indirect
	github.com/emicklei/go-restful/v3 v3.12.1 // indirect
	github.com/evanphx/json-patch v5.9.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.9.0 // indirect
	github.com/exponent-io/jsonpath v0.0.0-20151013193312-d6023ce2651d // indirect
	github.com/fatih/color v1.17.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/form3tech-oss/jwt-go v3.2.3+incompatible // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/go-errors/errors v1.4.2 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-logr/zapr v1.3.0 // indirect
	github.com/go-openapi/analysis v0.23.0 // indirect
	github.com/go-openapi/errors v0.22.0 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/loads v0.22.0 // indirect
	github.com/go-openapi/runtime v0.28.0 // indirect
	github.com/go-openapi/spec v0.21.0 // indirect
	github.com/go-openapi/strfmt v0.23.0 // indirect
	github.com/go-openapi/validate v0.24.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.19.0 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/gobuffalo/flect v1.0.2 // indirect
	github.com/gofrs/uuid/v5 v5.2.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.1 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/google/gnostic-models v0.6.9-0.20230804172637-c7be7c783f49 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20240424215950-a892ee059fd6 // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/googleapis/gax-go/v2 v2.13.0 // indirect
	github.com/gophercloud/gophercloud/v2 v2.0.0 // indirect
	github.com/gophercloud/utils/v2 v2.0.0-20240701101423-2401526caee5 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20190212212710-3befbb6ad0cc // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.7 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/itchyny/timefmt-go v0.1.6 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.4 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/jongio/azidext/go/azidext v0.5.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/kdomanski/iso9660 v0.2.1 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/metal3-io/baremetal-operator/apis v0.4.0 // indirect
	github.com/metal3-io/baremetal-operator/pkg/hardwareutils v0.4.0 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/microsoft/kiota-abstractions-go v0.18.0 // indirect
	github.com/microsoft/kiota-authentication-azure-go v0.6.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/moby/sys/mountinfo v0.7.1 // indirect
	github.com/moby/sys/user v0.1.0 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mxk/go-flowrate v0.0.0-20140419014527-cca7078d478f // indirect
	github.com/nutanix-cloud-native/cluster-api-provider-nutanix v1.5.3 // indirect
	github.com/nutanix-cloud-native/prism-go-client v0.5.0 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/opencontainers/runtime-spec v1.2.0 // indirect
	github.com/openshift/assisted-service/models v0.0.0 // indirect
	github.com/openshift/client-go v0.0.0-20240528061634-b054aa794d87 // indirect
	github.com/openshift/cloud-credential-operator v0.0.0-20240404165937-5e8812d64187 // indirect
	github.com/openshift/cluster-api-provider-baremetal v0.0.0-20220408122422-7a548effc26e // indirect
	github.com/openshift/cluster-api-provider-libvirt v0.2.1-0.20230308152226-83c0473d4429 // indirect
	github.com/openshift/cluster-api-provider-ovirt v0.1.1-0.20220323121149-e3f2850dd519 // indirect
	github.com/openshift/custom-resource-status v1.1.2 // indirect
	github.com/openshift/library-go v0.0.0-20240207105404-126b47137408 // indirect
	github.com/openshift/machine-api-operator v0.2.1-0.20240722145313-3a817c78946a // indirect
	github.com/openshift/machine-api-provider-gcp v0.0.1-0.20231014045125-6096cc86f3ba // indirect
	github.com/openshift/machine-api-provider-ibmcloud v0.0.0-20231207164151-6b0b8ea7b16d // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/ovirt/go-ovirt v0.0.0-20210809163552-d4276e35d3db // indirect
	github.com/pborman/uuid v1.2.0 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pierrec/lz4/v4 v4.1.17 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pkg/xattr v0.4.9 // indirect
	github.com/ppc64le-cloud/powervs-utils v0.0.0-20240610070307-1c0d75a5c247 // indirect
	github.com/prometheus/client_golang v1.19.0 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.52.2 // indirect
	github.com/prometheus/procfs v0.13.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/shurcooL/httpfs v0.0.0-20190707220628-8d4bc4ba7749 // indirect
	github.com/shurcooL/vfsgen v0.0.0-20181202132449-6a9ea43bcacd // indirect
	github.com/spf13/pflag v1.0.6-0.20210604193023-d5e0c0615ace // indirect
	github.com/syndtr/gocapability v0.0.0-20200815063812-42c35b437635 // indirect
	github.com/ulikunitz/xz v0.5.12 // indirect
	github.com/vincent-petithory/dataurl v1.0.0 // indirect
	github.com/vmware/govmomi v0.37.2 // indirect
	github.com/xlab/treeprint v1.2.0 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	go.mongodb.org/mongo-driver v1.14.0 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.49.0 // indirect
	go.opentelemetry.io/otel v1.24.0 // indirect
	go.opentelemetry.io/otel/metric v1.24.0 // indirect
	go.opentelemetry.io/otel/trace v1.24.0 // indirect
	go.starlark.net v0.0.0-20230525235612-a134d8f9ddca // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/exp v0.0.0-20240506185415-9bf2ced13842 // indirect
	golang.org/x/mod v0.19.0 // indirect
	golang.org/x/net v0.30.0 // indirect
	golang.org/x/oauth2 v0.23.0 // indirect
	golang.org/x/sync v0.9.0 // indirect
	golang.org/x/sys v0.27.0 // indirect
	golang.org/x/text v0.20.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.org/x/tools v0.22.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.4.0 // indirect
	google.golang.org/api v0.189.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240722135656-d784300faade // indirect
	google.golang.org/grpc v1.65.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/djherbis/times.v1 v1.3.0 // indirect
	gopkg.in/evanphx/json-patch.v5 v5.6.0 // indirect
	gopkg.in/gcfg.v1 v1.2.3 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gorm.io/gorm v1.24.5 // indirect
	k8s.io/apiextensions-apiserver v0.30.3 // indirect
	k8s.io/cli-runtime v0.30.3 // indirect
	k8s.io/client-go v0.30.3 // indirect
	k8s.io/cloud-provider-vsphere v1.30.1 // indirect
	k8s.io/component-base v0.30.3 // indirect
	k8s.io/klog v1.0.0 // indirect
	k8s.io/klog/v2 v2.120.1 // indirect
	k8s.io/kube-openapi v0.0.0-20240228011516-70dd3763d340 // indirect
	k8s.io/kubectl v0.30.3 // indirect
	k8s.io/utils v0.0.0-20240310230437-4693a0247e57 // indirect
	sigs.k8s.io/cluster-api v1.8.4 // indirect
	sigs.k8s.io/cluster-api-provider-aws/v2 v2.5.2-0.20240716201506-6e43273dd65d // indirect
	sigs.k8s.io/cluster-api-provider-azure v1.15.1-0.20240617212811-a52056dfb88c // indirect
	sigs.k8s.io/cluster-api-provider-gcp v1.7.1-0.20240724153512-c3b8b533143c // indirect
	sigs.k8s.io/cluster-api-provider-ibmcloud v0.7.0 // indirect
	sigs.k8s.io/cluster-api-provider-openstack v0.10.3 // indirect
	sigs.k8s.io/cluster-api-provider-vsphere v1.9.3 // indirect
	sigs.k8s.io/controller-runtime v0.18.5 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/kustomize/api v0.16.0 // indirect
	sigs.k8s.io/kustomize/kyaml v0.16.0 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.1 // indirect
)

replace (
	github.com/openshift/assisted-service/api => github.com/openshift/assisted-service/api v0.0.0-20230831114549-1922eda29cf8
	github.com/openshift/assisted-service/models => github.com/openshift/assisted-service/models v0.0.0-20230831114549-1922eda29cf8
	github.com/openshift/hive/apis => github.com/openshift/hive/apis v0.0.0-20220222213051-def9088fdb5a
	sigs.k8s.io/cluster-api-provider-ibmcloud => sigs.k8s.io/cluster-api-provider-ibmcloud v0.9.0-beta.0.0.20241029051454-9b0770491a76
)
