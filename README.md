### Source code for `27.6% of the Top 10 Million Sites are Dead`

Link: https://tonywang.io/blog/top-10-million-sites-27-percent-dead

### How to deploy the cluster:

1. Go to [page](https://spot.rackspace.com/ui/api-access/terraform) and generate a token for terraform
2. Save the token like `rackspace_spot_token = "token"` into `terraform.tfvars` of folder `terraform`
3. Run `make tf` to execute the terraform plan, it usually takes 30 minutes or longer for the cluster to be ready
4. Run `make redis` to deploy Redis cluster into your k8s cluster
5. Run `make d` to deploy all the required services into the k8s
