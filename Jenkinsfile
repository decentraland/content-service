node {
  stage('Git clone/update') {
        slackSend baseUrl: 'https://hooks.slack.com/services/', channel: '#pipeline-outputs', color: 'good', message: "Building  - ${env.JOB_NAME} ${env.BUILD_NUMBER} ${env.PROJECT} (<${env.BUILD_URL}|Open>)", teamDomain: 'decentralandteam', tokenCredentialId: 'slack-notification-pipeline-output'
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
  stage('Tes ting') {
        sh '''
          echo "Here goes the test"
        '''
  }
  stage('Image push') {
        sh '''
          docker push ${ECREGISTRY}/${PROJECT}:latest
          docker rmi ${ECREGISTRY}/${PROJECT}:latest
        '''
        }
  }
}
