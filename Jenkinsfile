def err = null

node {
  try {
    slackSend baseUrl: 'https://hooks.slack.com/services/', channel: '#pipeline-outputs', color: 'good', message: "Project - *${env.PROJECT}* \n\tStatus: *Started...*\n\tJob: *${env.JOB_NAME}*  \n\t Build Number: *${env.BUILD_NUMBER}* \n\tURL: (<${env.BUILD_URL}|Open>)", teamDomain: 'decentralandteam', tokenCredentialId: 'slack-notification-pipeline-output'
    stage('Git clone/update') {
          sshagent(credentials : ['content-service']) {
          sh '''
              #Check the content of the payload and extract the Branch
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
              docker ps -a | awk '{ print $1 }' | grep -v CONTAINER | xargs docker rm
            fi
          '''
    }
    stage('Testing') {
          sh '''
            cd ${PROJECT}
            docker run -d --name content_service_redis -p 6379:6379 --rm redis:4.0.11
            docker run -d --name content_service_golang -p 8000:8000 --rm ${ECREGISTRY}/${PROJECT}:latest
            if test $? -ne 0; then
              echo "ERROR!!, `docker logs content_service_golang`"
              exit 2
            fi
            ./demo.sh
            curl 'http://localhost:8000/mappings'   -F 'metadata={"value": "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn","signature": "0x96a6e3f69b25fcf89d5af9fb9d6f17da8dd86548f486822e74296af1d8bcaf920e67684e2a15cd942526a4ede10dd5483eccb381d92f88b932858d7a466f99ed1b","pubKey": "0xa08a656ac52c0b32902a76e122d2973b022caa0e","validityType": 0,"validity": "2018-12-12T14:49:14.074000000Z","sequence": 2}'   -F 'QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn=[{"cid": "QmaiT7TzzKVjgJ6PJnovQn9DYrFcFyLnFaBseMdyLHCtX8","name": "assets/"},{"cid": "QmbdQuGbRFZdeqmK3PJyLV3m4p2KDELKRS4GfaXyehz672","name": "assets/test.txt"},{"cid": "QmbGdhmRstTdbNBKxqVbGpjiPxy2A5nqrDLuk9KFmQtwox","name": "build.json"},{"cid": "QmTBetsUR4WC1fUB3oM7sDCBQZiHXrsp4LXarqTnHFZ9on","name": "package.json"},{"cid": "QmfRoY2437YZgrJK9s5Vvkj6z9xH4DqGT1VKp1WFoh6Ec4","name": "scene.json"},{"cid": "QmSXv3Qgr8pjoYNXZqMhE5Lo9f8FXpYF5cN7vndXsYqJou","name": "scene.tsx"},{"cid": "Qmdv1drP1dkNFKjX6YqL91Go4mY141ZSFQy311qidk9HJc","name": "tsconfig.json"}]'   -F 'QmbdQuGbRFZdeqmK3PJyLV3m4p2KDELKRS4GfaXyehz672=@demo/assets/test.txt'   -F 'QmbGdhmRstTdbNBKxqVbGpjiPxy2A5nqrDLuk9KFmQtwox=@demo/build.json'   -F 'QmTBetsUR4WC1fUB3oM7sDCBQZiHXrsp4LXarqTnHFZ9on=@demo/package.json'   -F 'QmfRoY2437YZgrJK9s5Vvkj6z9xH4DqGT1VKp1WFoh6Ec4=@demo/scene.json'   -F 'QmSXv3Qgr8pjoYNXZqMhE5Lo9f8FXpYF5cN7vndXsYqJou=@demo/scene.tsx'   -F 'Qmdv1drP1dkNFKjX6YqL91Go4mY141ZSFQy311qidk9HJc=@demo/tsconfig.json'
            wget http://localhost:8000/contents/QmbdQuGbRFZdeqmK3PJyLV3m4p2KDELKRS4GfaXyehz672 -O /dev/null
            if test $? -ne 0; then
              echo "ERROR!!, `curl http://localhost:8000/contents/QmbdQuGbRFZdeqmK3PJyLV3m4p2KDELKRS4GfaXyehz672`"
              exit 2
            fi
          '''
    }
    stage('Image push') {
          sh '''
            docker push ${ECREGISTRY}/${PROJECT}:latest
            docker rmi ${ECREGISTRY}/${PROJECT}:latest
          '''
    }
    slackSend baseUrl: 'https://hooks.slack.com/services/', channel: '#pipeline-outputs', color: 'good', message: "Project - *${env.PROJECT}* \n\tStatus: *Finished OK*\n\tJob: *${env.JOB_NAME}*  \n\t Build Number: *${env.BUILD_NUMBER}* \n\tURL: (<${env.BUILD_URL}|Open>)", teamDomain: 'decentralandteam', tokenCredentialId: 'slack-notification-pipeline-output'
  } catch (caughtError) { //End of Try
    err = caughtError
    slackSend baseUrl: 'https://hooks.slack.com/services/', channel: '#pipeline-outputs', color: '#FF0000', message: "Project - *${env.PROJECT}* \n\tError: ${err}\n\tJob: *${env.JOB_NAME}*  \n\t Build Number: *${env.BUILD_NUMBER}* \n\tURL: (<${env.BUILD_URL}|Open>)", teamDomain: 'decentralandteam', tokenCredentialId: 'slack-notification-pipeline-output'
    currentBuild.result = "FAILURE"
  }
}
