FROM lawrencegripper/sfonebox
RUN apt-get install nodejs -y && apt-get clean
RUN sed -i "s%IPAddr=.*%IPAddr=localhost%g" ClusterDeployer.sh