import argparse
import json
import time
import opentracing

from io import StringIO
from jaeger_client import Config

def init_jaeger_tracer(service_name='python-job'):
    config = Config(config={
        'sampler': {
            'type': 'const',
            'param': 1,
        },
        'local_agent': {
            'reporting_host': 'localhost',
            'reporting_port': '6831',
        },
        'logging': True,
    }, service_name=service_name, validate=True)
    return config.initialize_tracer()

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='span args')
    parser.add_argument('-s', "--span", type=str, help="span context")
    args = parser.parse_args()
    if args.span is None:
        exit(1)

    # unmarshal span context carrier from json string
    io = StringIO(args.span)
    text_carrier = json.load(io)
    print(text_carrier)

    tracer = init_jaeger_tracer()
    
    # unwind span context from carrier
    span_context = opentracing.tracer.extract(opentracing.Format.TEXT_MAP, text_carrier)
    print(span_context)
    with tracer.start_span('ChildSpan', child_of=span_context) as child_span:
        print(child_span)
        child_span.log_kv({'event': 'done'})
    time.sleep(2)   # flush the spans - https://github.com/jaegertracing/jaeger-client-python/issues/50
    tracer.close()  # flush any buffered spans