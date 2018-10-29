def err = null
def ECREGISTRYDEV = "872049612737.dkr.ecr.us-east-1.amazonaws.com"
def ECREGISTRYPROD = "245402993223.dkr.ecr.us-east-1.amazonaws.com"

node {
  try {
    slackSend baseUrl: 'https://hooks.slack.com/services/', channel: '#pipeline-outputs', color: 'good', message: "Project/Branch - *${env.JOB_NAME}* \n\tStatus: *Started...*  \n\tBuild Number: *${env.BUILD_NUMBER}* \n\tURL: (<${env.BUILD_URL}|Open>)", teamDomain: 'decentralandteam', tokenCredentialId: 'slack-notification-pipeline-output'
    stage('Build Image') {
          sshagent(credentials : ['content-service']) {
          sh '''
          case ${BRANCH_NAME} in
            master) ECREGISTRY=${ECREGISTRYPROD}
            ;;
            *) ECREGISTRY=${ECREGISTRYDEV}
            ;;
          esac
          aws ecr get-login --no-include-email | bash
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
