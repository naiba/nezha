import os
import time
import requests
import hashlib
from github import Github


def get_github_latest_release():
    g = Github()
    repo = g.get_repo("naiba/nezha")
    release = repo.get_latest_release()
    if release:
        print(f"Latest release tag is: {release.tag_name}")
        print(f"Latest release info is: {release.body}")
        files = []
        for asset in release.get_assets():
            url = asset.browser_download_url
            name = asset.name

            response = requests.get(url)
            if response.status_code == 200:
                with open(name, 'wb') as f:
                    f.write(response.content)
                print(f"Downloaded {name}")
            else:
                print(f"Failed to download {name}")
            file_abs_path = get_abs_path(asset.name)
            files.append(file_abs_path)
        sync_to_gitee(release.tag_name, release.body, files)
    else:
        print("No releases found.")


def sync_to_gitee(tag: str, body: str, files: slice):
    release_id = ""
    owner = "naibahq"
    repo = "nezha"
    release_api_uri = f"https://gitee.com/api/v5/repos/{owner}/{repo}/releases"
    api_client = requests.Session()
    api_client.headers.update({
        'Accept': 'application/json',
        'Content-Type': 'application/json'
    })

    access_token = os.environ['GITEE_TOKEN']
    release_data = {
        'access_token': access_token,
        'tag_name': tag,
        'name': tag,
        'body': body,
        'prerelease': False,
        'target_commitish': 'master'
    }
    release_api_response = api_client.post(release_api_uri, json=release_data)
    if release_api_response.status_code == 201:
        release_info = release_api_response.json()
        release_id = release_info.get('id')
    else:
        print(
            f"Request failed with status code {release_api_response.status_code}")

    print(f"Gitee release id: {release_id}")
    asset_api_uri = f"{release_api_uri}/{release_id}/attach_files"

    for file_path in files:
        files = {
            'file': open(file_path, 'rb')
        }

        asset_api_response = requests.post(
            asset_api_uri, params={'access_token': access_token}, files=files)

        if asset_api_response.status_code == 201:
            asset_info = asset_api_response.json()
            asset_name = asset_info.get('name')
            print(f"Successfully uploaded {asset_name}!")
        else:
            print(
                f"Request failed with status code {asset_api_response.status_code}")

    api_client.close()
    print("Sync is completed!")


def get_abs_path(path: str):
    wd = os.getcwd()
    return os.path.join(wd, path)


get_github_latest_release()
