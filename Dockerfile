FROM mirror.gcr.io/library/golang:1.18 as builder

WORKDIR /workspace

# Copy full workspace for git version info
COPY . .

# Build
RUN hack/build-binaries.sh

# Create image
FROM registry.ci.openshift.org/ocp/4.12:tools
RUN yum install -y skopeo gdisk && \
    yum clean -y all
RUN curl -sL https://mirror.openshift.com/pub/openshift-v4/clients/ocp-dev-preview/latest/oc-mirror.tar.gz \
    | tar xvzf - -C /usr/local/bin && \
    chmod +x /usr/local/bin/oc-mirror

COPY run.sh /run.sh
COPY help.sh /help.sh

COPY --from=builder /workspace/_output/ /usr/local/bin/

# no tool is more important than others
ENTRYPOINT ["/run.sh"]
CMD ["/help.sh"]
