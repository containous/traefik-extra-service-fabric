FROM servicefabricoss/service-fabric-onebox
WORKDIR /home/ClusterDeployer
RUN ./setup.sh
RUN locale-gen en_US.UTF-8
ENV LANG=en_US.UTF-8
ENV LANGUAGE=en_US:en
ENV LC_ALL=en_US.UTF-8
EXPOSE 19080 19000 80 443
CMD /etc/init.d/ssh start && ./run.sh

