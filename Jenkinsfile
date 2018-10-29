def err = null

node {
  try {
    slackSend baseUrl: 'https://hooks.slack.com/services/', channel: '#pipeline-outputs', color: 'good', message: "Project - *${env.JOB_NAME}* \n\tBranch: *${env.BRANCH_NAME}* \n\tStatus: *Started...*\n\tJob: *${env.JOB_NAME}*  \n\tBuild Number: *${env.BUILD_NUMBER}* \n\tURL: (<${env.BUILD_URL}|Open>)", teamDomain: 'decentralandteam', tokenCredentialId: 'slack-notification-pipeline-output'
    stage('Git clone/update') {
          sshagent(credentials : ['content-service']) {
          sh '''
              echo "Hello World"
              '''
          }
    }
    slackSend baseUrl: 'https://hooks.slack.com/services/', channel: '#pipeline-outputs', color: 'good', message: "Project - *${env.JOB_NAME}* \n\tBranch: *${env.BRANCH_NAME}* \n\tStatus: *Finished OK*\n\tJob: *${env.JOB_NAME}*  \n\t Build Number: *${env.BUILD_NUMBER}* \n\tURL: (<${env.BUILD_URL}|Open>)", teamDomain: 'decentralandteam', tokenCredentialId: 'slack-notification-pipeline-output'
  } catch (caughtError) { //End of Try
    err = caughtError
    slackSend baseUrl: 'https://hooks.slack.com/services/', channel: '#pipeline-outputs', color: '#FF0000', message: "Project - *${env.JOB_NAME}* \n\tError: ${err}\n\tJob: *${env.JOB_NAME}*  \n\t Build Number: *${env.BUILD_NUMBER}* \n\tURL: (<${env.BUILD_URL}|Open>)", teamDomain: 'decentralandteam', tokenCredentialId: 'slack-notification-pipeline-output'
    currentBuild.result = "FAILURE"
  }
}
