FROM python:3
RUN pip3 install sfctl
RUN sfctl cluster select --endpoint http://localhost:19080
WORKDIR /src
ENTRYPOINT [ "bash" ]