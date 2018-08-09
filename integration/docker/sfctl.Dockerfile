FROM python:3
RUN pip install sfctl
WORKDIR /src
ENTRYPOINT [ "bash" ]
