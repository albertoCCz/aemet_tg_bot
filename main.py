import os
import logging
import json
from datetime import datetime

from telegram import Update
from telegram.ext import filters, MessageHandler, ApplicationBuilder, ContextTypes, CommandHandler, Job

from html_scrapper import MyHTMLParser, get_url_html


logging.basicConfig(
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    level=logging.INFO
)

# ---------- config
TEST = False
BOT_TOKEN = os.environ['BOT_TOKEN']
CHAT_IDS = {
    'TEST': os.environ['CHAT_ID_TEST'],
    'A1': os.environ['CHAT_ID_A1'],
    'A2': os.environ['CHAT_ID_A2'],
    'C1': os.environ['CHAT_ID_C1']
}
START_MSSG = 'Os mantendré informados!'
TEMPLATE = './templates/template.txt'
PDF_LISTS_PATH = './pdfs-registry'
AEMET_URLS = {
    # LABEL     : URL
    'TEST': {
        'Libre'  : 'https://www.aemet.es/es/empleo_y_becas/empleo_publico/oposiciones/grupo_a1/acceso_libre/acceso_libre_2021_2022',
    },
    'A1': {
        'Libre'  : 'https://www.aemet.es/es/empleo_y_becas/empleo_publico/oposiciones/grupo_a1/acceso_libre/acceso_libre_2021_2022',
        'Interna': 'https://www.aemet.es/es/empleo_y_becas/empleo_publico/oposiciones/grupo_a1/promocion_interna/acceso_interna_2021_2022',
    },
    'A2': {
        'Libre'  : 'https://www.aemet.es/es/empleo_y_becas/empleo_publico/oposiciones/grupo_a2/acceso_libre/acceso_libre_2021_2022',
        'Interna': 'https://www.aemet.es/es/empleo_y_becas/empleo_publico/oposiciones/grupo_a2/promocion_interna/acceso_interna_2021_2022',
    },
    'C1': {
        'Libre'  : 'https://www.aemet.es/es/empleo_y_becas/empleo_publico/oposiciones/grupo_c1/acceso_libre/acceso_libre_2021_2022',
    }
}
TIME_INTERVAL = 30    # in seconds
# ----------


def get_updated_pdfs_mssg(category: str, pdf_name: str, pdf_data: dict) -> str:
    with open(TEMPLATE, 'r') as f:
        mssg = f.read()
        
    mssg = mssg.replace('{pdf_name}', pdf_name)
    mssg = mssg.replace('{category}', category)
    mssg = mssg.replace('{pdf_url}', pdf_data['pdf_url'])
    mssg = mssg.replace('{pdf_date}', pdf_data['pdf_date'])
    return mssg

async def start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await context.bot.send_message(chat_id=update.effective_chat.id, text=START_MSSG)

async def echo(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await context.bot.send_message(chat_id=update.effective_chat.id, text=update.message.text)

async def scrap_coordinator(context: ContextTypes.DEFAULT_TYPE):
    print(f"\nSCRAP COORDINATOR: TEST={TEST}\n")
    if TEST:
        for group in AEMET_URLS.keys():
            if group == 'TEST':
                for category, url in AEMET_URLS[group].items():
                    print(f"\nCALLING scrap_pdfs with group={group}, category={category}, url={url}\n")
                    await scrap_pdfs(context, group=group, category=category, url=url)
    else:
        for group in AEMET_URLS.keys():
            if group != 'TEST':
                for category, url in AEMET_URLS[group].items():
                    await scrap_pdfs(context, group=group, category=category, url=url)
    
async def scrap_pdfs(context, group: str, category: str, url: str):
    # get 'new' list of pdfs
    page = get_url_html(url)
    parser = MyHTMLParser()
    parser.feed(page)
    pdfs = parser.get_pdfs()

    pdf_file_path = f"{PDF_LISTS_PATH}/pdfs-list-{group}-{category.replace(' ', '-')}.json"

    # open 'old' list of pdfs
    try:
        with open(pdf_file_path, 'r') as f:
            old_pdfs = json.load(f)
    except:
        print("No previous pdfs file found.")
        old_pdfs = None

    updated_pdfs = {}
    if old_pdfs:
        for pdf_name, pdf_data in pdfs.items():
            if pdf_name in old_pdfs:
                try: # try to parse date found for new pdf
                    new_pdf_date = datetime.strptime(pdf_data['pdf_date'], '%d/%m/%Y')
                except Exception as exe:
                     print(f"PDF: {pdf_name} for category {category} could not be processed: {exe}")
                     
                old_pdf_date = datetime.strptime(old_pdfs[pdf_name]['pdf_date'], '%d/%m/%Y')
                if new_pdf_date > old_pdf_date:
                    updated_pdfs.update({pdf_name: pdf_data})
            else:
                updated_pdfs.update({pdf_name: pdf_data})
    else:
        updated_pdfs = pdfs    

    if len(updated_pdfs) > 0:
        for pdf_name, pdf_data in updated_pdfs.items():
            with open(pdf_file_path, 'w+') as f:
                json.dump(pdfs, f)

            mssg = get_updated_pdfs_mssg(category, pdf_name, pdf_data)
            if TEST:
                print(f"\nCHAT ID:  {CHAT_IDS[group]}\n")
            await context.bot.send_message(chat_id=CHAT_IDS[group], text=mssg, parse_mode='HTML')


if __name__ == '__main__':
    application = ApplicationBuilder().token(BOT_TOKEN).build()
    job_queue = application.job_queue
    
    start_handler = CommandHandler('start', start)
    application.add_handler(start_handler)

    job_scrap = job_scrap_pdfs = job_queue.run_repeating(scrap_coordinator, interval=TIME_INTERVAL, first=1)
    
    application.run_polling()
    
