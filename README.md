    python3 -m venv venv
    source venv/bin/activate
    python -m pip install --upgrade pip
    python -m pip install -r requirements.txt


    python -m pytest --disable-pytest-warnings



    docker run  -e PULUMI_ACCESS_TOKEN=<token> dirien/minecraft-automationapi destroy my-minecraft-sever -s ediri/dev