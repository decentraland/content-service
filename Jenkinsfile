def err = null

node {
  try {
    slackSend baseUrl: 'https://hooks.slack.com/services/', channel: '#pipeline-outputs', color: 'good', message: "Project/Branch - *${env.JOB_NAME}* \n\tStatus: *Started...*  \n\tBuild Number: *${env.BUILD_NUMBER}* \n\tURL: (<${env.BUILD_URL}|Open>)", teamDomain: 'decentralandteam', tokenCredentialId: 'slack-notification-pipeline-output'
    stage('Clone repo') {
          sshagent(credentials : ['content-service']) {
          sh '''
          #Retrieveing the job name. This is used as the first part of the image name
          PROJECT=`echo ${JOB_NAME} | awk -F/ '{ print $1 }'`
          REPOURL="git@github.com:decentraland"

          #Verifying from which registry shall pull the Image, depending on the branch
          case ${BRANCH_NAME} in
            master) ECREGISTRY="245402993223.dkr.ecr.us-east-1.amazonaws.com"
            ;;
            *) ECREGISTRY="872049612737.dkr.ecr.us-east-1.amazonaws.com"
            ;;
          esac
          git clone ${REPOURL}/${PROJECT}.git && cd ${PROJECT} || cd ${PROJECT}
          git fetch
          git pull
          git checkout ${BRANCH_NAME}
          '''
    }
    stage('Build Image') {
          sshagent(credentials : ['content-service']) {
          sh '''
          #Retrieveing the job name. This is used as the first part of the image name
          PROJECT=`echo ${JOB_NAME} | awk -F/ '{ print $1 }'`
          REPOURL="git@github.com:decentraland"

          #Verifying from which registry shall pull the Image, depending on the branch
          test -h ${JENKINS_HOME}/.aws && unlink ${JENKINS_HOME}/.aws
          case ${BRANCH_NAME} in
            master) ECREGISTRY="245402993223.dkr.ecr.us-east-1.amazonaws.com"
            ln -s ${JENKINS_HOME}/.aws-prod ${JENKINS_HOME}/.aws
            ;;
            *) ECREGISTRY="872049612737.dkr.ecr.us-east-1.amazonaws.com"
            ln -s ${JENKINS_HOME}/.aws-dev ${JENKINS_HOME}/.aws
            ;;
          esac

          aws ecr get-login --no-include-email | bash
          #So far, the last image is tagged as latest.
          #This must change to commit number
          cd ${PROJECT}
          docker build -t ${ECREGISTRY}/${PROJECT}:latest .
          '''
          }
    }
    slackSend baseUrl: 'https://hooks.slack.com/services/', channel: '#pipeline-outputs', color: 'good', message: "Project/Branch - *${env.JOB_NAME}* \n\tStatus: *Finished OK*  \n\t Build Number: *${env.BUILD_NUMBER}* \n\tURL: (<${env.BUILD_URL}|Open>)", teamDomain: 'decentralandteam', tokenCredentialId: 'slack-notification-pipeline-output'
  } catch (caughtError) { //End of Try
    err = caughtError
    slackSend baseUrl: 'https://hooks.slack.com/services/', channel: '#pipeline-outputs', color: '#FF0000', message: "Project/Branch - *${env.JOB_NAME}* \n\tError: ${err}  \n\t Build Number: *${env.BUILD_NUMBER}* \n\tURL: (<${env.BUILD_URL}|Open>)", teamDomain: 'decentralandteam', tokenCredentialId: 'slack-notification-pipeline-output'
    currentBuild.result = "FAILURE"
  }
}
