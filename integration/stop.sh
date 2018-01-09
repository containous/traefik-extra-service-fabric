
#!/bin/bash
echo "######## Remove previous containers if they exist ###########"
docker rm -f sftestcluster 
docker rm -f sfsampleinstaller 
docker rm -f sfappinstaller