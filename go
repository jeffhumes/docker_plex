#!/bin/bash
export BASE_DIR=`pwd`

# import external configuration file entries
. ./go.conf

export APP_UID=$(id -u)
export APP_GID=$(id -g)
export WARS_DIR="${BASE_DIR}/wars"
export CONF_DIR="${BASE_DIR}/conf"
export PROPS_DIR="${BASE_DIR}/properties"
export LOGS_DIR="${BASE_DIR}/logs"
export CERTS_DIR="${BASE_DIR}/certs"
export KEY_DIR="/home/dockrusr/.ssh"
export PERSIST_DIR="${BASE_DIR}/persist"



usage(){
	echo "Usage:"
	echo " $0 <option>"
	echo "    start		- starts the container in the background"
	echo "    start-int	- builds the container in interactive mode"
	echo "    restart	- restarts the container"
	echo "    stop		- stops the container"
	echo "    copy-files	- copy required files"
	echo "    update-ports	- update the standard port numbers in the configs to your selected ports"
	echo "    shell		- creates a shell in the container to look around"
	echo "    build		- build the docker image that will be run (reads Dockerfile)"
	exit 0
}

create_required_directories(){
	if [ ! -d ${CONF_DIR} ]
	then
		echo "Creating local conf directory: ${CONF_DIR}"
		mkdir ${CONF_DIR}
	fi

	if [ ! -d ${WARS_DIR} ]
	then
		echo "Creating local warfile directory: ${WARS_DIR}"
		mkdir ${WARS_DIR}
	fi

	if [ ! -d "${LOGS_DIR}" ]; then
		echo "Creating local logging directory: ${LOGS_DIR}"
      		mkdir "${LOGS_DIR}"
	fi


	if [ ! -d "${PROPS_DIR}" ]; then
		echo "Creating local properties directory: ${PROPS_DIR}"
      		mkdir "${PROPS_DIR}"
	fi

	if [ ! -d "${CERTS_DIR}" ]; then
		echo "Creating local certs directory: ${CERTS_DIR}"
      		mkdir "${CERTS_DIR}"
	fi

}

get_required_files(){
	if [ ${TOMCAT_INSTALL} = "true" ]; then

		SERVER_XML="${CONF_DIR}/server.xml"
		if [ ! -f "${SERVER_XML}" ]; then
			cp -av dist/server.xml ${SERVER_XML}
 		fi
	
		TOMCAT_USERS_XML="${CONF_DIR}/tomcat-users.xml"
		if [ ! -f ${TOMCAT_USERS_XML} ]
		then
			cp -av dist/tomcat-users.xml ${TOMCAT_USERS_XML}
			
		fi
	
		MANAGER_CONTEXT_XML="${CONF_DIR}/manager-context.xml"
		if [ ! -f ${MANAGER_CONTEXT_XML} ]
		then
			cp -av dist/manager-context.xml ${MANAGER_CONTEXT_XML}
		fi
	
		MANAGER_WEB_XML="${CONF_DIR}/manager-web.xml"
		if [ ! -f ${MANAGER_WEB_XML} ]
		then
			cp -av dist/manager-web.xml ${MANAGER_WEB_XML}
		fi
	
		MAVEN_SETTINGS_XML="${CONF_DIR}/maven-settings.xml"
			if [ ! -f ${MAVEN_SETTINGS_XML} ]
		then
			cp -av dist/maven-settings.xml ${MAVEN_SETTINGS_XML}
		fi
	
	fi
}

generate_ssl_key(){
	if [ ${TOMCAT_INSTALL} = "true" ]; then

		KEYSTORE_FILE="${CERTS_DIR}/keystore.jks"

		if [ ! -f ${KEYSTORE_FILE} ]
		then
			echo "No keystore found, creating a self-signed keystore just to get us up and running... YOU WILL NEED TO UPDATE WITH A REAL SSL KEY"
			docker run -ti \
			-v ${CERTS_DIR}:/vm-certs \
			${INSTANCE_NAME} \
			keytool -genkey \
			-keystore /vm-certs/keystore.jks \
			-keyalg RSA \
			-keysize 2048 \
			-validity 10000 \
			-alias app \
			-dname "cn='temp.itg.ti.com' o='Texas Instruments', ou='IT', st='Texas',  c='US'" \
			-storepass Ry9coo2J \
			-keypass Ry9coo2J 
		fi
	fi
}

#keytool -genkey -keystore keystore.ss -keyalg RSA -keysize 2048         -validity 10000 -alias app -dname 'cn="docker-eet.itg.ti.com", o="Texas Instruments", ou="IT", st="Texas",  c="US"'         -storepass Ry9coo2J -keypass Ry9coo2J

generate_start_option_lines(){
(
        for MAPPING in `cat go_mappings.conf | grep -v ^#`
        do 
                TYPE=`echo ${MAPPING} | awk -F \| '{print $1}'`

                if [ ${TYPE} = "VOLUME" ]; then
			VOLUME_TYPE=`echo ${MAPPING} | awk -F \| '{print $1}'`
			HOST_LOCATION=`echo ${MAPPING} | awk -F \| '{print $3}'`
			
			if [ ${VOLUME_TYPE} = 'FILE' ]; then
				if [ ! -f ${HOST_LOCATION} ]; then
					echo "${HOST_LOCATION} of type ${VOLUME_TYPE} not found!, exiting..."	
					exit 991;
				fi  
			elif [ ${VOLUME_TYPE} = 'DIRECTORY' ]; then
				if [ ! -d ${HOST_LOCATION} ]; then
					echo "${HOST_LOCATION} of type ${VOLUME_TYPE} not found!, exiting..."	
					exit 991;
				fi  
			fi		

			GUEST_LOCATION=`echo ${MAPPING} | awk -F \| '{print $4}'`
                        echo "-v ${HOST_LOCATION}:${GUEST_LOCATION} "

                fi

                if [ ${TYPE} = "WARFILE" ]; then
			HOST_LOCATION=`echo ${MAPPING} | awk -F \| '{print $2}'`
			GUEST_LOCATION=`echo ${MAPPING} | awk -F \| '{print $3}'`
                        echo "-v ${HOST_LOCATION}:${GUEST_LOCATION} "

                fi

                if [ ${TYPE} = "PORT" ]; then
			HOST_LOCATION=`echo ${MAPPING} | awk -F \| '{print $2}'`
			GUEST_LOCATION=`echo ${MAPPING} | awk -F \| '{print $3}'`
                        echo "-p ${HOST_LOCATION}:${GUEST_LOCATION} "
		fi


		
        done
        cd ${BASE_DIR}
)
}

get_wars(){
	if [ ${TOMCAT_INSTALL} = "true" ]; then
		if [ ! -f "${WARS_DIR}/ffw-properties-manager.war" ]; then
			echo "${WARS_DIR}/ffw-properties-manager.war does not exist... Getting it from artifactory..."
			cd ${WARS_DIR} && \
			curl -O "https://artifactory.itg.ti.com/artifactory/generic-tmg-prod-local/pmo/sa/docker-demo/ffw-properties-manager.war" && \

			if [ $? -eq 0 ]
			then
				cd ${BASE_DIR}/
			else
				echo "ERROR:  could not retrieve ffw-properties-manager.war"
				exit 44
			fi
		fi
	fi
}

change_ports(){
	 util_scripts/change_ports
}

remove_git(){
	if [ -d .git ]; then
		if [ ${REMOVE_GIT} = "true" ]; then
			echo "Removing git directory to stop any changes here making it to bitbucket"
			rm -rf .git
		fi
	fi
}

case "$1" in
  start)
   create_required_directories
   get_required_files
   generate_ssl_key
   get_wars
   remove_git

	GENERATED_OPTIONS=$(generate_start_option_lines);
	echo ${GENERATED_OPTIONS}

	if [ ${TESTING_GO_FILE} = "true" ]; then
		echo "Not starting instance due to TESTING_GO_FILE set to true"
	else
	docker run -d \
	--user ${IMAGE_USER}  \
	-v /etc/localtime:/etc/localtime:ro \
	-e CATALINA_OPTS="-Dffw.home=/usr/local/tomcat/properties -DJENKINS_HOME=/usr/local/tomcat/jenkins/" \
	-e JAVA_OPTS="-Duser.timezone=UTC" \
	${GENERATED_OPTIONS} \
	--name ${INSTANCE_NAME} \
	--hostname ${INSTANCE_NAME} \
  	--restart=${DOCKER_RESTART_POLICY} \
   	${INSTANCE_NAME}
	fi
  ;;
  start-int)
   create_required_directories
   get_required_files
   generate_ssl_key
   get_wars
   remove_git

	GENERATED_OPTIONS=$(generate_start_option_lines);
	echo ${GENERATED_OPTIONS}

	if [ ${TESTING_GO_FILE} = "true" ]; then
		echo "Not starting instance due to TESTING_GO_FILE set to true"
	else
	docker run -ti \
	--user ${IMAGE_USER}  \
	-v /etc/localtime:/etc/localtime:ro \
	-e CATALINA_OPTS="-Dffw.home=/usr/local/tomcat/properties -DJENKINS_HOME=/usr/local/tomcat/jenkins/" \
	-e JAVA_OPTS="-Duser.timezone=UTC" \
	${GENERATED_OPTIONS} \
	--name ${INSTANCE_NAME} \
	--hostname ${INSTANCE_NAME} \
  	--restart=${DOCKER_RESTART_POLICY} \
   	${INSTANCE_NAME}
	fi
  ;;
  exec)
    docker exec -ti ${INSTANCE_NAME} ${@:2}
  ;;
  root)
    docker exec -u root -ti ${INSTANCE_NAME} bash 
  ;;
  logs)
    docker logs ${INSTANCE_NAME} 2>&1
  ;;
  tail)
    docker logs --follow -n1 ${INSTANCE_NAME} 2>&1
  ;;
  #simple stop/start
  restart)
    $0 stop
    $0 start
  ;;
  stop)
    docker stop ${INSTANCE_NAME};
    docker rm ${INSTANCE_NAME};
  ;;
  build)
	echo "User: ${IMAGE_USER}"
	echo "User Home: ${IMAGE_USER_HOME}"
	echo "User Shell: ${IMAGE_USER_SHELL}"
	echo "Chrome Remote Desktop Install? ${CHROME_REMOTE}"

	if [ ${INSTANCE_NAME} = "CHANGE_ME" ]; then
		echo "-----------------------------------------------------------"
		echo "Please update the file go.conf with your settings first !!!"
		echo "-----------------------------------------------------------"
		exit 41
	elif  ! [ -f Dockerfile ]; then
		if [ ${TOMCAT_INSTALL} = "true" ]; then
			ln -s templates/Dockerfile_host_wars Dockerfile;
		else
			ln -s templates/Dockerfile_no_tomcat  Dockerfile;
		fi
	fi
	
	grep WELL_KNOWN_HTTP Dockerfile > /dev/null
	if [ ${?} = 0 ]; then
		echo "-----------------------------------------------------------"
                echo "Please run ./go update-ports prior to running build !!!"
                echo "-----------------------------------------------------------"
                exit 42
	fi

if [ ${DOCKER_BUILD_VERBOSE} = "true" ]; then
	docker build  --rm \
	--tag ${INSTANCE_NAME} \
	--build-arg UID=`id -u` \
	--build-arg GID=`id -g` \
	--build-arg USR=${IMAGE_USER} \
	--build-arg IMG_USR_HOME=${IMAGE_USER_HOME} \
	--build-arg IMG_USR_SHELL=${IMAGE_USER_SHELL} \
	--build-arg INSTALL_TYPE=${INSTALL_TYPE} \
	--build-arg COMPANY=${COMPANY} \
	--build-arg CHROME_REMOTE=${CHROME_REMOTE} \
	--build-arg CHROME_REMOTE_DESKTOP=${CHROME_REMOTE_DESKTOP} \
	--build-arg OS_UPDATES=${OS_UPDATES} \
	--build-arg EXTRA_PACKAGES=${EXTRA_PACKAGES} \
	--build-arg INSTANCE_NAME=${INSTANCE_NAME} \
	--load \
	.
else
	docker build  --rm \
	--tag ${INSTANCE_NAME} \
	--build-arg UID=`id -u` \
	--build-arg GID=`id -g` \
	--build-arg USR=${IMAGE_USER} \
	--build-arg IMG_USR_HOME=${IMAGE_USER_HOME} \
	--build-arg IMG_USR_SHELL=${IMAGE_USER_SHELL} \
	--build-arg INSTALL_TYPE=${INSTALL_TYPE} \
	--build-arg COMPANY=${COMPANY} \
	--build-arg CHROME_REMOTE=${CHROME_REMOTE} \
	--build-arg CHROME_REMOTE_DESKTOP=${CHROME_REMOTE_DESKTOP} \
	--build-arg OS_UPDATES=${OS_UPDATES} \
	--build-arg EXTRA_PACKAGES=${EXTRA_PACKAGES} \
	--build-arg INSTANCE_NAME=${INSTANCE_NAME} \
	--load \
	--quiet \
	.
fi
  ;;
  copy-files)
   	create_required_directories
   	get_required_files
  ;;
  update-ports)
	change_ports
  ;;
  *)
	usage
  ;;
esac

