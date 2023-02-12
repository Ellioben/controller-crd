# controller-crd
use k8scrd by controller

# init

参考sampla-controller，创建pkg/apis/group/v1

> You can copy from sampla-controller

add doc.go,regester.go,types.go and document hack

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

init you hack script

```bash
vim update-codegen.sh
```

