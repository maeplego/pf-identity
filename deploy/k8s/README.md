# P01 Kubernetes manifests

`pf-cloud-k8s` の `portfolio-integration` overlay から参照される IdP / admin の Deployment です。

```powershell
# 単体では apply しない。連携デモは pf-cloud-k8s から:
cd ..\..\pf-cloud-k8s
.\scripts\up.ps1
```

イメージ tag は overlay の `images:` で上書きします（既定 `pf-identity-*:latest`）。
