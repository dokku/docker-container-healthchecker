FROM golang:1.22.5-bookworm

# hadolint ignore=DL3027
RUN apt-get update \
    && apt install apt-transport-https build-essential curl gnupg2 jq lintian rsync rubygems-integration ruby-dev ruby -qy \
    && git clone https://github.com/bats-core/bats-core.git /tmp/bats-core \
    && /tmp/bats-core/install.sh /usr/local \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# hadolint ignore=DL3028
RUN gem install --quiet rake fpm package_cloud

WORKDIR /src

RUN curl -fsSLO https://download.docker.com/linux/static/stable/x86_64/docker-20.10.14.tgz && tar --strip-components=1 -xvzf docker-20.10.14.tgz -C /usr/local/bin
