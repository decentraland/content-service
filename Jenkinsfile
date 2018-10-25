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
            sleep 120
            docker logs content_service_golang
            echo " ------------------------------------ "
            echo "| Executing demo routine....         |"
            echo " ------------------------------------ "
            ./demo.sh
            curl 'http://127.0.0.1:8000/mappings'   -F 'metadata={"value": "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn","signature": "0x96a6e3f69b25fcf89d5af9fb9d6f17da8dd86548f486822e74296af1d8bcaf920e67684e2a15cd942526a4ede10dd5483eccb381d92f88b932858d7a466f99ed1b","pubKey": "0xa08a656ac52c0b32902a76e122d2973b022caa0e","validityType": 0,"validity": "2018-12-12T14:49:14.074000000Z","sequence": 2}'   -F 'QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn=[{"cid": "QmaiT7TzzKVjgJ6PJnovQn9DYrFcFyLnFaBseMdyLHCtX8","name": "assets/"},{"cid": "QmbdQuGbRFZdeqmK3PJyLV3m4p2KDELKRS4GfaXyehz672","name": "assets/test.txt"},{"cid": "QmbGdhmRstTdbNBKxqVbGpjiPxy2A5nqrDLuk9KFmQtwox","name": "build.json"},{"cid": "QmTBetsUR4WC1fUB3oM7sDCBQZiHXrsp4LXarqTnHFZ9on","name": "package.json"},{"cid": "QmfRoY2437YZgrJK9s5Vvkj6z9xH4DqGT1VKp1WFoh6Ec4","name": "scene.json"},{"cid": "QmSXv3Qgr8pjoYNXZqMhE5Lo9f8FXpYF5cN7vndXsYqJou","name": "scene.tsx"},{"cid": "Qmdv1drP1dkNFKjX6YqL91Go4mY141ZSFQy311qidk9HJc","name": "tsconfig.json"}]'   -F 'QmbdQuGbRFZdeqmK3PJyLV3m4p2KDELKRS4GfaXyehz672=@demo/assets/test.txt'   -F 'QmbGdhmRstTdbNBKxqVbGpjiPxy2A5nqrDLuk9KFmQtwox=@demo/build.json'   -F 'QmTBetsUR4WC1fUB3oM7sDCBQZiHXrsp4LXarqTnHFZ9on=@demo/package.json'   -F 'QmfRoY2437YZgrJK9s5Vvkj6z9xH4DqGT1VKp1WFoh6Ec4=@demo/scene.json'   -F 'QmSXv3Qgr8pjoYNXZqMhE5Lo9f8FXpYF5cN7vndXsYqJou=@demo/scene.tsx'   -F 'Qmdv1drP1dkNFKjX6YqL91Go4mY141ZSFQy311qidk9HJc=@demo/tsconfig.json'
            echo " ------------------------------------ "
            echo "| Attempting to download....         |"
            echo " ------------------------------------ "
            if test $? -ne 0; then
              echo "ERROR!!, `curl http://localhost:8000/contents/QmbdQuGbRFZdeqmK3PJyLV3m4p2KDELKRS4GfaXyehz672`"
              docker stop content_service_redis content_service_golang
              exit 2
            fi
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
