pipeline {
    agent any

    stages {
        stage('Build') {
            steps {
                sh 'docker build -f dockerfile.app1 -t my-go-app:${BUILD_NUMBER} -t my-go-app:latest .'
            }
        }

        stage('Deploy') {
            steps {
                withCredentials([file(credentialsId: 'app1-env-file', variable: 'ENV_FILE')]) {
                    sh '''
                        # Stop and remove old container if it exists
                        docker stop test-api || true
                        docker rm test-api || true

                        # Run the new one
                        docker run -d \
                          --name test-api \
                          -p 9090:8080 \
                          --env-file $ENV_FILE \
                          --restart unless-stopped \
                          my-go-app:latest
                    '''
                }
            }
        }

        stage('Smoke Test') {
            steps {
                sh '''
                    sleep 5
                    curl -f http://localhost:9090/health || (docker logs test-api && exit 1)
                '''
            }
        }
    }

    post {
        failure {
            sh 'docker logs test-api || true'
        }
    }
}