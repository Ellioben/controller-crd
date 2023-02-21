# Control custom resources in kubenetes.
# 1. Usage
1. Definded the resource for kubenetes
2. Using custom reource by k8scontroller
3. Practice & Reference sample-controller

# 2. Init env & code

reference kubenetes/sampla-controller，create pkg/apis/group/v1
```go
mkdir pkg/apis/group/v1
```

> You can copy from sampla-controller

Added doc.go,register.go,types.go and document hack

```
.
├── LICENSE
├── README.md
├── go.mod
├── go.sum
├── hack
│   ├── boilerplate.go.txt
│   ├── custom-boilerplate.go.txt
│   ├── tools.go
│   ├── update-codegen.sh
│   └── verify-codegen.sh
└── pkg
    └── apis
        └── groupkind
            ├── register.go
            └── v1alpha1
                ├── doc.go
                ├── register.go
                └── types.go
```

# config your crd's script

# 3. init you hack script

```bash
vim update-codegen.sh
```

1. you need to revise the update-codegen.sh
2. go mod vendor
3. ./update-codegen.sh   **（As you can see, this is based on code-generate）**

```bash
"${CODEGEN_PKG}/generate-groups.sh" "deepcopy,client,informer,lister" \


# Usage: $(basename "$0") <generators> <output-package> <apis-package> <groups-versions> ...

  <generators>        the generators comma separated to run (deepcopy,defaulter,client,lister,informer) or "all".
  <output-package>    the output package name (e.g. github.com/example/project/pkg/generated).
  <apis-package>      the external types dir (e.g. github.com/example/api or github.com/example/project/pkg/apis).
  <groups-versions>   the groups and their versions in the format "groupA:v1,v2 groupB:v1 groupC:v2", relative
                      to <api-package>.
  ...                 arbitrary flags passed to all generator binaries.


Examples:
  $(basename "$0") all             github.com/example/project/pkg/client github.com/example/project/pkg/apis "foo:v1 bar:v1alpha1,v1beta1"
  $(basename "$0") deepcopy,client github.com/example/project/pkg/client github.com/example/project/pkg/apis "foo:v1 bar:v1alpha1,v1beta1"
```

# 4. Generate code for custom resources

generate in you pkg

```bash
./hack/update-codegen.sh  
Generating deepcopy funcs
Generating clientset for groupkind:v1alpha1 at controller-crd/pkg/generated/clientset
Generating listers for groupkind:v1alpha1 at controller-crd/pkg/generated/listers
Generating informers for groupkind:v1alpha1 at controller-crd/pkg/generated/informers


tree -L 3

└── pkg
    ├── apis
    │   └── groupkind
    │       ├── register.go
    │       └── v1alpha1
    │           ├── doc.go
    │           ├── register.go
    │           ├── types.go
    │           └── zz_generated.deepcopy.go
    └── generated
        ├── clientset
        │   └── versioned
        │       ├── clientset.go
        │       ├── fake
        │       ├── scheme
        │       └── typed
        ├── informers
        │   └── externalversions
        │       ├── factory.go
        │       ├── generic.go
        │       ├── groupkind
        │       └── internalinterfaces
        └── listers
            └── groupkind
                └── v1alpha1


```



# 5. Generate mainfest

```bash
controller-gen crd paths=./...output:crd:dir=config/crd
```

