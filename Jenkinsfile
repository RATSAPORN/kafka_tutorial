pipeline {
    agent any

    stages {
        stage('Deploy') {
            steps {
                withCredentials([file(credentialsId: 'app1-env-file', variable: 'ENV_FILE')]) {
                    sh '''
                        # Clean up any leftover standalone test container from before
                        docker stop test-api || true
                        docker rm test-api || true

                        # Compose reads a literal ".env" file for variable substitution
                        cp "$ENV_FILE" .env

                        # Build and (re)start just the app1 service
                        docker compose build app1
                        docker compose up -d app1
                    '''
                }
            }
        }

        stage('Smoke Test') {
            steps {
                sh '''
                    sleep 10
                    curl -f http://localhost:8080/health || (docker compose logs app1 --tail 100 && exit 1)
                '''
            }
        }
    }

    post {
        failure {
            sh 'docker compose logs app1 --tail 100 || true'
        }
        always {
            sh 'rm -f .env || true'
        }
    }
}