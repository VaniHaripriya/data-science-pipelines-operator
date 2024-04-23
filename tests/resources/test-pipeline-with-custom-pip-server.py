from kfp import dsl, compiler


@dsl.component(base_image="quay.io/opendatahub/ds-pipelines-ci-executor-image:v1.0",
packages_to_install=['boto3'],
pip_index_urls=['http://pypi-server.test-pypiserver.svc.cluster.local:8080'])
def say_hello(name: str) -> str:
    hello_text = f'Hello, {name}!'
    print(hello_text)
    return hello_text


@dsl.pipeline
def hello_pipeline(recipient: str) -> str:
    hello_task = say_hello(name=recipient)
    return hello_task.output


if __name__ == '__main__':
    compiler.Compiler().compile(hello_pipeline, __file__ + '.yaml')
