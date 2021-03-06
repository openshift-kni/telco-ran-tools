FROM fedora:latest
RUN dnf install -y skopeo gdisk util-linux xfsprogs parted && \
    dnf clean -y all
RUN curl -sL https://mirror.openshift.com/pub/openshift-v4/clients/ocp-dev-preview/latest/oc-mirror.tar.gz | tar xvzf - -C /usr/local/bin && chmod +x /usr/local/bin/oc-mirror
COPY _output /usr/local/bin
COPY run.sh /run.sh
COPY help.sh /help.sh
# no tool is more important than others
ENTRYPOINT ["/run.sh"]
CMD ["/help.sh"]
