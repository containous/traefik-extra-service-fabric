FROM python:3
RUN pip3 install sfctl
WORKDIR /src
ENTRYPOINT [ "bash" ]