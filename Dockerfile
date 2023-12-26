FROM kasmweb/core-ubuntu-focal:1.14.0
USER root
ENV USER root

ENV HOME /home/kasm-default-profile
ENV STARTUPDIR /dockerstartup
ENV INST_SCRIPTS $STARTUPDIR/install
WORKDIR $HOME

######### Customize Container Here ###########

LABEL maintainer="jhumes@ti.com"

SHELL ["/bin/bash", "-c"]

ENV OS_UPDATES true
ENV EXTRA_PACKAGES true
ENV INSTALL_KDENLIVE true

RUN if [[ "${OS_UPDATES}" = "true" ]] || [[ "${EXTRA_PACKAGES}" = "true" ]]; then apt-get update; fi
RUN if [[ "${OS_UPDATES}" = "true" ]]; then apt-get -y upgrade; fi
RUN if [[ "${EXTRA_PACKAGES}" = "true" ]]; then apt-get -y install sudo vim git curl; fi
RUN if [[ "${EXTRA_PACKAGES}" = "true" ]]; then sed -i 's/%sudo.*$/%sudo  ALL=(ALL) NOPASSWD: ALL/g' /etc/sudoers; fi

RUN if [[ "${USER}" != "root"  ]]; then groupadd -g "${GID}" "${USR}"; else noop=1; fi
RUN if [[ "${USER}" != "root"  ]]; then useradd -u "${UID}" -g "${GID}" -m -d ${IMG_USR_HOME} -s ${IMG_USR_SHELL} ${USR}; else noop=1; fi
RUN if [[ "${USER}" != "root" ]]; then usermod -aG sudo ${USR}; fi

RUN if [[ "${INSTALL_KDENLIVE}" = "true" ]]; then add-apt-repository --yes ppa:kdenlive/kdenlive-stable && apt update && apt install -y kdenlive; fi
RUN if [[ "${INSTALL_KDENLIVE}" = "true" ]]; then apt update && apt install -y vlc; fi

######### End Customizations ###########

RUN chown 1000:0 $HOME
RUN $STARTUPDIR/set_user_permission.sh $HOME

ENV HOME /home/kasm-user
WORKDIR $HOME
RUN mkdir -p $HOME && chown -R 1000:0 $HOME

USER 1000

