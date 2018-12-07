import requests
import json
import datetime
import time


def mag_evaluate(query, wait_time=10):
    try:
        response = requests.get(
            url=
            "http://api.labs.cognitive.microsoft.com/academic/v1.0/evaluate",
            params={
                "expr": query,
                "model": "latest",
                "count": "10",
                "offset": "0",
                #"orderby": "null",
                "attributes": "Id,Ti,Y,CC,RId,D,AA.AuN",
            },
            headers={
                "Ocp-Apim-Subscription-Key":
                "665fd999adc04d7488ea810daf3dde75",
                "subscription-key": "",
            },
        )

        if response.status_code == 429:
            # print("Rate limited: waiting {} seconds".format(wait_time))
            time.sleep(wait_time)
            return mag_evaluate(query, wait_time / 2)
        wait_time = 1

        # print('Response HTTP Status Code: {status_code}'.format(
        #     status_code=response.status_code))

        resp = json.loads(response.content)
        if 'entities' in resp:
            for obj in resp['entities']:
                obj.pop('E', None)
            return resp['entities']
        else:
            return resp
    except requests.exceptions.RequestException:
        print('HTTP Request failed')
        return None


def mag_get_title(mag_id):
    obj = mag_evaluate("Id={}".format(mag_id))[0]
    return str(obj['Ti']) + " (" + str([str(auth['AuN']) for auth in obj['AA']]) + ", " + str(int(obj['Y'])) + ")"


format_str = '%Y-%m-%d'  # The format


def mag_get_date(mag_id):
    obj = mag_evaluate("Id={}".format(mag_id))[0]
    return datetime.datetime.strptime(obj['D'], format_str)
