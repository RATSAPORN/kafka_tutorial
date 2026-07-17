pipeline {
    agent any   // runs on the Jenkins host, which has Docker installed

    environment {
        IMAGE_NAME = 'my-go-app'
        IMAGE_TAG  = "${env.BUILD_NUMBER}"
    }

    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Build & Test (via Docker)') {
            steps {
                sh 'docker build -t ${IMAGE_NAME}:${IMAGE_TAG} .'
            }
        }

        stage('Tag as latest') {
            steps {
                sh 'docker tag ${IMAGE_NAME}:${IMAGE_TAG} ${IMAGE_NAME}:latest'
            }
        }

        // Uncomment once you have a registry to push to (Docker Hub, ECR, GHCR, etc.)
        /*
        stage('Push') {
            steps {
                withCredentials([usernamePassword(credentialsId: 'dockerhub-creds', usernameVariable: 'USER', passwordVariable: 'PASS')]) {
                    sh 'echo $PASS | docker login -u $USER --password-stdin'
                    sh 'docker push ${IMAGE_NAME}:${IMAGE_TAG}'
                    sh 'docker push ${IMAGE_NAME}:latest'
                }
            }
        }
        */
    }

    post {
        success {
            echo "Built ${IMAGE_NAME}:${IMAGE_TAG}"
        }
        failure {
            echo 'Pipeline failed — check the stage that broke above.'
        }
        always {
            sh 'docker image prune -f'   // clean up dangling layers so WSL disk doesn't fill up
        }
    }
}