terraform import rpaas_cert_manager.resource_name "service::instance::issuer"
# issuer == <resource name>.<resource kind>.<resource group>

# example
terraform import rpaas_cert_manager.mycertmanager "rpaasv2-be::my-rpaas::issuer.ClusterIssuer.example.com"