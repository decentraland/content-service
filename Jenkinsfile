def err = null

node {
  try {
    slackSend baseUrl: 'https://hooks.slack.com/services/', channel: '#pipeline-outputs', color: 'good', message: "Project - *${env.PROJECT}* \n\tBranch: *${Branch}* \n\tStatus: *Started...*\n\tJob: *${env.JOB_NAME}*  \n\tBuild Number: *${env.BUILD_NUMBER}* \n\tURL: (<${env.BUILD_URL}|Open>)", teamDomain: 'decentralandteam', tokenCredentialId: 'slack-notification-pipeline-output'
    stage('Git clone/update') {
          sshagent(credentials : ['content-service']) {
          sh '''
              #Check the  content of the payload and extract the Branch
              Branch=`echo $Branch | awk -F"/" '{print $NF}'`
              git clone ${REPOURL}/${PROJECT}.git && cd ${PROJECT} || cd ${PROJECT}
              git checkout $Branch
              if test $? -ne 0; then
                echo "Unable to checkout $Branch."
                fi
              git fetch
              git pull'''
            }
    }
    stage('Image building') {
      sh '''
            aws ecr get-login --no-include-email | bash
            cd ${PROJECT}
            docker build -t ${ECREGISTRY}/${PROJECT}:latest .
      '''
    }
    stage('Removing  previous containers') {
          sh '''
            RUNNING_CONTAINERS=`docker ps | awk '{ print $1 }' | grep -v CONTAINER | wc -l`
            if test ${RUNNING_CONTAINERS} -ne 0; then
              docker ps | awk '{ print $1 }' | grep -v CONTAINER | xargs docker stop
            fi
            RUNNING_CONTAINERS=`docker ps -a | awk '{ print $1 }' | grep -v CONTAINER | wc -l`
            if test ${RUNNING_CONTAINERS} -ne 0; then
              docker ps -a | awk '{ print $1 }' | grep -v CONTAINER | xargs docker rm -f
            fi
          '''
    }
    stage('Testing') {
          sh '''
            cd ${PROJECT}
            echo " ------------------------------------------ "
            echo "| Starting redis....         |"
            echo " ------------------------------------------ "
            docker run -d --name content_service_redis -p 6379:6379 --rm redis:4.0.11
            echo " ----------------------------- "
            echo "| starting golang....         |"
            echo " ----------------------------- "
            docker run -d --name content_service_golang -p 8000:8000 --rm ${ECREGISTRY}/${PROJECT}:latest
            if test $? -ne 0; then
              echo "ERROR!!, `docker logs content_service_golang`"
              docker stop content_service_redis content_service_golang
              exit 2
            fi
            echo " ------------------------------------------ "
            echo "| Waiting for container startup....         |"
            echo " ------------------------------------------ "
            docker logs content_service_golang
            docker stop content_service_redis content_service_golang
          '''
    }
    stage('Image push') {
          sh '''
            echo " ------------------------------------------ "
            echo "| Waiting for container to finish....         |"
            echo " ------------------------------------------ "
            docker push ${ECREGISTRY}/${PROJECT}:latest
            docker rmi -f ${ECREGISTRY}/${PROJECT}:latest
          '''
    }
    stage('Launching Deploy') {
          sh '''
            echo " ------------------------------------------ "
            echo "| Launching deploy job....         |"
            echo " ------------------------------------------ "
            Branch=`echo $Branch | awk -F"/" '{print $NF}'`
            case $Branch in
              master)
                      cd ${PROJECT}
                      git checkout master
                      test -h ${JENKINS_HOME}/.aws && unlink ${JENKINS_HOME}/.aws
                      ln -s ${JENKINS_HOME}/.aws-prod ${JENKINS_HOME}/.aws
                      cd .terraform/main
                      ./terraform-run.sh us-east-1 prod master
              ;;

              development)
                      cd ${PROJECT}
                      git checkout development
                      test -h ${JENKINS_HOME}/.aws && unlink ${JENKINS_HOME}/.aws
                      ln -s ${JENKINS_HOME}/.aws-dev ${JENKINS_HOME}/.aws
                      cd .terraform/main
                      ./terraform-run.sh us-east-1 dev development
              ;;

              *)
                      cd ${PROJECT}
                      git checkout $Branch
                      test -h ${JENKINS_HOME}/.aws && unlink ${JENKINS_HOME}/.aws
                      ln -s ${JENKINS_HOME}/.aws-dev ${JENKINS_HOME}/.aws
                      cd .terraform/main
                      ./terraform-run.sh us-east-1 dev $Branch
              ;;
            esac
          '''
    }
    slackSend baseUrl: 'https://hooks.slack.com/services/', channel: '#pipeline-outputs', color: 'good', message: "Project - *${env.PROJECT}* \n\tBranch: *${Branch}* \n\tStatus: *Finished OK*\n\tJob: *${env.JOB_NAME}*  \n\t Build Number: *${env.BUILD_NUMBER}* \n\tURL: (<${env.BUILD_URL}|Open>)", teamDomain: 'decentralandteam', tokenCredentialId: 'slack-notification-pipeline-output'
  } catch (caughtError) { //End of Try
    err = caughtError
    slackSend baseUrl: 'https://hooks.slack.com/services/', channel: '#pipeline-outputs', color: '#FF0000', message: "Project - *${env.PROJECT}* \n\tError: ${err}\n\tJob: *${env.JOB_NAME}*  \n\t Build Number: *${env.BUILD_NUMBER}* \n\tURL: (<${env.BUILD_URL}|Open>)", teamDomain: 'decentralandteam', tokenCredentialId: 'slack-notification-pipeline-output'
    currentBuild.result = "FAILURE"
  }
}
