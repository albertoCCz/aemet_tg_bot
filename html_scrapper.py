import requests
import re
import json
from html.parser import HTMLParser
from datetime import datetime
from pprint import PrettyPrinter

pprint = PrettyPrinter(indent=4, depth=4, sort_dicts=False)


def get_url_html(url: str) -> str:
    r = requests.get(url)
    assert r.status_code == 200, f"Error: Request Status Code is '{r.status_code}'"
    
    return r.text


def get_uniques(xs: list) -> list:
    uniques = []
    for x in xs:
        if x not in uniques:
            uniques.append(x)
    return uniques


es_months = ["enero",
             "febrero",
             "marzo",
             "abril",
             "mayo",
             "junio",
             "julio",
             "agosto",
             "septiembre",
             "octubre",
             "noviembre",
             "diciembre"]


def es_month_to_number(month: str) -> int:
    return es_months.index(month) + 1

    
class MyHTMLParser(HTMLParser):
    def __init__(self):
        super().__init__()
        self.pdf_url_prefix = 'https://www.aemet.es'
        self.pdfs: dict = {}
        self.last_pdf: str = ''
        self.listen_for_date: bool = False
        self.listen_for_name: bool = False
    
    def handle_starttag(self, tag, attrs):
        if tag == 'a':
            for attr in attrs:
                if len(attr) > 1 and attr[0] == 'href' and attr[1].endswith('.pdf'):
                    pdf_url = attr[1]
                    self.pdfs.update({pdf_url: {
                        'name': None,
                        'date': None
                    }})
                    self.last_pdf = pdf_url
                    self.listen_for_name = True
                    self.listen_for_date = True

    def handle_data(self, data):
        if self.listen_for_name:
            match = re.findall(r"\([0-9]{1,10} [MK]B\)", data)
            if match:
                pdf_name = data
                self.pdfs[self.last_pdf]['name'] = pdf_name
                self.listen_for_name = False
        
        if self.listen_for_date:
            match = re.findall(r"[0-9]{1,2} de [a-z]{1,10} de[l]{0,1} [0-9]{3,4}", data)
            if match:
                match = match[0]
                if 'del' in match:
                    match = match.replace('del', 'de')
                day, month, year = match.split(' de ')
                try:
                    possible_date = f"{year}/{es_month_to_number(month):02}/{day}"
                    date = datetime.strptime(possible_date, "%Y/%m/%d")
                except ValueError:
                    date = datetime(2000, 1, 1)
                    # TODO: handle bad formatted dates in a better way, probably notifying users...
                    print(f"\nWarning: Could not parse date '{possible_date}' - bad formatted, for pdf '{self.last_pdf}'. For now I will ignore it...\n")
                    
                self.pdfs[self.last_pdf]['date'] = date.strftime("%Y/%m/%d")
                self.listen_for_date = False

    def __format_pdfs_found(self):
        if len(self.pdfs) > 0:
            pdfs = {}
            for pdf_url, pdf_info in self.pdfs.items():
                pdfs.update({pdf_info['name']: {
                    'pdf_url': self.pdf_url_prefix + pdf_url,
                    'pdf_date': pdf_info['date']
                }})
    
            self.pdfs = pdfs

    def get_pdfs(self) -> list:
        self.__format_pdfs_found()
        return self.pdfs
                                 

if __name__ == '__main__':
    aemet_url = 'https://www.aemet.es/es/empleo_y_becas/empleo_publico/oposiciones/grupo_a1/acceso_libre/acceso_libre_2021_2022'
    page = get_url_html(aemet_url)
    parser = MyHTMLParser()
    parser.feed(page)
    pdfs = parser.get_pdfs()
    print(pdfs)
    print("Number of pdfs found:", len(pdfs))
    
