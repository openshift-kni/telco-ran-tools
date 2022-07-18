FROM fedora:latest
RUN dnf install -y skopeo gdisk util-linux xfsprogs parted && \
    dnf clean -y all
COPY _output /usr/local/bin
COPY run.sh /run.sh
COPY help.sh /help.sh
# no tool is more important than others
ENTRYPOINT ["/run.sh"]
CMD ["/help.sh"]
