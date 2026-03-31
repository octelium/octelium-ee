# Octelium Enterprise

Welcome to the official repository for Octelium Enterprise. This repository houses the enterprise-grade features for [Octelium](https://github.com/octelium/octelium).

## Commercial & Production Use

Octelium Enterprise is designed to support organizations and production environments. If you are deploying Octelium in a commercial or production setting (as defined within the `LICENSE` file), an active enterprise license is required. 

* **Learn More & Purchase:** Discover our feature tiers and acquire a license by visiting [Octelium Enterprise](https://octelium.com/enterprise).
* **Get in Touch:** For enterprise support, custom inquiries, or general questions, please reach out via our [contact form](https://octelium.com/contact) or email us directly at [contact@octelium.com](mailto:contact@octelium.com).

## License

This repository is distributed under the **OCTELIUM ENTERPRISE SOURCE-AVAILABLE LICENSE**. Please refer to the `LICENSE` file located in the root directory for the complete legal text and terms of use.

**Personal Use:** Octelium Enterprise is completely free of charge, forever, for non-commercial personal use (as strictly defined within the `LICENSE` file).


## Installation

On your already installed and running Octelium _Cluster_ (see the _Cluster_ quick installation guide [here](https://octelium.com/docs/octelium/latest/overview/quick-install) or the production installation guide [here](https://octelium.com/docs/octelium/latest/install/cluster/installing-cluster)), you can install the Octelium enterprise package for that _Cluster_ via the `octops install-package` command (you need to have `octops`, which you can install [here](https://octelium.com/docs/octelium/latest/install/cli/install), version `v0.29.0` or later) as follows:

```bash
octops install-package <DOMAIN> --package octeliumee
```

You can override that location via the `--kubeconfig` flag (by default is located at `$HOME/.kube/config`) as follows:

```bash
./bin/octops install-package <DOMAIN> --package octeliumee --kubeconfig </PATH/TO/KUNECONFIG>
```


You can also install a specific version as follows:


```bash
octops install-package <DOMAIN> --package octeliumee --version 1.2.3
```

And you can upgrade the Octelium enterprise package as follows:

```bash
octops install-package <DOMAIN> --package octeliumee --upgrade
```