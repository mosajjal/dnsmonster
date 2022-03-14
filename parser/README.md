This folder contains an Azure Sentinel ASIM DNS parser for DNSmonster

Feature reference: https://docs.microsoft.com/en-us/azure/sentinel/normalization-about-parsers
Schema reference: https://docs.microsoft.com/en-us/azure/sentinel/dns-normalization-schema
ASIM deployment: https://aka.ms/DeployASIM

Usage:
With ASIM being a preview feature, the documentation is challenging, but the basic steps are:
- Deploy the DnsDeploymentCustomUnifyingParsers.json to your Azure tenant (https://github.com/Azure/Azure-Sentinel/tree/master/ASIM/deploy/EmptyCustomUnifyingParsers)
- Deploy the ASimDnsDnsMoster.json template  OR
  Create ASimDnsDnsMonster and vimDnsDnsMonster functions in your log analytics workspace and then modify the ASimDns and imDns functions as per https://docs.microsoft.com/en-us/azure/sentinel/normalization-manage-parsers#manage-workspace-deployed-unifying-parsers

See also : https://docs.microsoft.com/en-us/azure/sentinel/normalization-manage-parsers

Additional DNS parsers can be found here: https://github.com/Azure/Azure-Sentinel/tree/master/Parsers/ASimDns