FROM lawrencegripper/sfonebox
RUN apt-get install nodejs -y
RUN sed -i "s%IPAddr=.*%IPAddr=localhost%g" ClusterDeployer.sh